/**
 * Audit / compliance endpoints.
 *
 * GET  /audit/history?keyId=<id>         Decrypt history visible under a stored key
 * POST /audit/history                    Decrypt history from a raw key (not stored)
 * GET  /audit/export?keyId=<id>&fmt=csv  Export as CSV or JSON
 * GET  /audit/stealth                    Generate a stealth address for a researcher
 *
 * These endpoints are the integration surface for the audit dashboard and for
 * third-party auditors who receive a viewing key out-of-band.
 */

import { Router, Request, Response } from "express";
import { z } from "zod";
import { v4 as uuidv4 } from "uuid";
import { CloakClient } from "../cloak-client.js";
import { store } from "../store.js";
import { AuditHistoryResult, TransactionRecord } from "../types.js";

const RawKeySchema = z.object({
  key: z.string().min(1),
  label: z.string().default("External audit"),
});

function toCsv(result: AuditHistoryResult): string {
  const header = "id,timestamp,type,mint,amount_usdc,tx_signature,sender,recipient,finding_id";
  const rows = result.transactions.map((t: TransactionRecord) =>
    [
      t.id,
      t.timestamp,
      t.type,
      t.mint,
      (t.amountUsdc / 1_000_000).toFixed(6),
      t.txSignature,
      t.senderShielded ?? "",
      t.recipientShielded ?? "",
      t.findingId ?? "",
    ].join(",")
  );
  return [header, ...rows].join("\n");
}

export function auditRouter(cloak: CloakClient): Router {
  const router = Router();

  // Decrypt history using a stored viewing key ID
  router.get("/history", async (req: Request, res: Response) => {
    const keyId = req.query.keyId as string;
    if (!keyId) {
      return res.status(400).json({ error: "keyId query parameter required" });
    }

    const record = store.getViewingKey(keyId);
    if (!record) {
      return res.status(404).json({ error: "Viewing key not found" });
    }

    try {
      const result = await cloak.buildAuditHistory(record);
      return res.json(result);
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  // Decrypt history from a raw viewing key supplied in the body
  // (for third-party auditors who receive the key out-of-band)
  router.post("/history", async (req: Request, res: Response) => {
    const parsed = RawKeySchema.safeParse(req.body);
    if (!parsed.success) {
      return res.status(400).json({ error: parsed.error.flatten() });
    }

    const ephemeralRecord = {
      id: uuidv4(),
      key: parsed.data.key,
      scope: "full" as const,
      label: parsed.data.label,
      createdAt: new Date().toISOString(),
      findingIds: [] as string[],
    };

    try {
      const result = await cloak.buildAuditHistory(ephemeralRecord);
      return res.json(result);
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  // Export — CSV or JSON
  router.get("/export", async (req: Request, res: Response) => {
    const keyId = req.query.keyId as string;
    const fmt = (req.query.fmt as string) ?? "json";

    if (!keyId) {
      return res.status(400).json({ error: "keyId query parameter required" });
    }

    const record = store.getViewingKey(keyId);
    if (!record) {
      return res.status(404).json({ error: "Viewing key not found" });
    }

    try {
      const result = await cloak.buildAuditHistory(record);

      if (fmt === "csv") {
        res.setHeader("Content-Type", "text/csv");
        res.setHeader(
          "Content-Disposition",
          `attachment; filename="cloak-audit-${keyId.slice(0, 8)}.csv"`
        );
        return res.send(toCsv(result));
      }

      res.setHeader(
        "Content-Disposition",
        `attachment; filename="cloak-audit-${keyId.slice(0, 8)}.json"`
      );
      return res.json(result);
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  // Generate a stealth address for a researcher to receive a shielded payout
  router.get("/stealth", async (_req: Request, res: Response) => {
    try {
      const addr = await cloak.generateStealthAddress();
      return res.json({
        stealthAddress: addr,
        note: "Share this address with the payer. Your wallet will never appear on-chain.",
      });
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  return router;
}
