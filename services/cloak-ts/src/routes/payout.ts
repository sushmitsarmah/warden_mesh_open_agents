/**
 * POST /payout
 *
 * Executes a private batch bug-bounty disbursement via Cloak's shielded pool.
 * One transaction fans out to all recipients; amounts and addresses are hidden.
 * Optionally issues a viewing key so the protocol's finance team can audit later.
 *
 * Request body: PayoutRequest (see types.ts)
 * Response:     PayoutResult
 */

import { Router } from "express";
import { z } from "zod";
import { v4 as uuidv4 } from "uuid";
import { CloakClient } from "../cloak-client.js";
import { store } from "../store.js";
import { PayoutResult, ViewingKeyRecord } from "../types.js";

const PayoutSchema = z.object({
  findingId: z.string().min(1),
  recipients: z
    .array(
      z.object({
        stealthAddress: z.string().min(1),
        amountUsdc: z.number().int().positive(),
        label: z.string().optional(),
      })
    )
    .min(1),
  mint: z.enum(["USDC", "USDT", "SOL"]).default("USDC"),
  generateViewingKey: z.boolean().default(true),
  viewingKeyLabel: z.string().optional(),
  viewingKeyScope: z.enum(["full", "amounts_only", "time_limited"]).default("full"),
  viewingKeyExpiresAt: z.string().datetime().optional(),
});

export function payoutRouter(cloak: CloakClient): Router {
  const router = Router();

  router.post("/", async (req, res) => {
    const parsed = PayoutSchema.safeParse(req.body);
    if (!parsed.success) {
      return res.status(400).json({ error: parsed.error.flatten() });
    }

    const body = parsed.data;
    const existing = store.getPayout(body.findingId);
    if (existing) {
      return res.status(409).json({
        error: "Payout already executed for this finding",
        payout: existing,
      });
    }

    try {
      const txSignature = await cloak.batchDisbursement(body.recipients, body.mint);

      const totalAmountUsdc = body.recipients.reduce(
        (sum, r) => sum + r.amountUsdc,
        0
      );

      let viewingKey: ViewingKeyRecord | undefined;

      if (body.generateViewingKey) {
        const expiresAt = body.viewingKeyExpiresAt
          ? new Date(body.viewingKeyExpiresAt)
          : undefined;

        const { rawKey } = await cloak.generateViewingKey(
          body.viewingKeyScope,
          body.viewingKeyLabel ?? `Bounty payout — finding ${body.findingId}`,
          expiresAt
        );

        viewingKey = {
          id: uuidv4(),
          key: rawKey,
          scope: body.viewingKeyScope,
          label: body.viewingKeyLabel ?? `Bounty payout — finding ${body.findingId}`,
          createdAt: new Date().toISOString(),
          expiresAt: body.viewingKeyExpiresAt,
          findingIds: [body.findingId],
        };

        store.saveViewingKey(viewingKey);
      }

      const result: PayoutResult = {
        txSignature,
        findingId: body.findingId,
        totalAmountUsdc,
        recipientCount: body.recipients.length,
        viewingKey,
        shieldedAt: new Date().toISOString(),
      };

      store.savePayout(result);

      return res.status(201).json(result);
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  // GET /payout/:findingId — retrieve payout result
  router.get("/:findingId", (req, res) => {
    const result = store.getPayout(req.params.findingId);
    if (!result) {
      return res.status(404).json({ error: "Payout not found" });
    }
    // Strip the raw viewing key from the response — only the ID is returned
    const { viewingKey, ...safe } = result;
    return res.json({
      ...safe,
      viewingKeyId: viewingKey?.id,
    });
  });

  // GET /payout — list all payouts (no raw keys)
  router.get("/", (_req, res) => {
    const payouts = store.listPayouts().map(({ viewingKey, ...p }) => ({
      ...p,
      viewingKeyId: viewingKey?.id,
    }));
    return res.json(payouts);
  });

  return router;
}
