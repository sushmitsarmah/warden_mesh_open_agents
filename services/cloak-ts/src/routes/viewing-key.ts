/**
 * Viewing key management endpoints.
 *
 * POST /viewing-key               Issue a new key (standalone, not tied to a payout)
 * GET  /viewing-key               List all key records (IDs + metadata, never raw keys)
 * GET  /viewing-key/:id           Get key record by ID (no raw key)
 * GET  /viewing-key/:id/raw       Return raw key string (protected — add auth in prod)
 * DELETE /viewing-key/:id         Revoke a key
 *
 * Raw keys are only returned from /raw so callers can decide when to transmit them.
 * In production, /raw should require the caller to prove ownership of the protocol
 * that originally requested the key (e.g. via a signed challenge).
 */

import { Router } from "express";
import { z } from "zod";
import { v4 as uuidv4 } from "uuid";
import { CloakClient } from "../cloak-client.js";
import { store } from "../store.js";
import { ViewingKeyRecord } from "../types.js";

const IssueSchema = z.object({
  scope: z.enum(["full", "amounts_only", "time_limited"]).default("full"),
  label: z.string().min(1),
  findingIds: z.array(z.string()).default([]),
  expiresAt: z.string().datetime().optional(),
});

export function viewingKeyRouter(cloak: CloakClient): Router {
  const router = Router();

  router.post("/", async (req, res) => {
    const parsed = IssueSchema.safeParse(req.body);
    if (!parsed.success) {
      return res.status(400).json({ error: parsed.error.flatten() });
    }

    const body = parsed.data;
    try {
      const expiresAt = body.expiresAt ? new Date(body.expiresAt) : undefined;
      const { rawKey } = await cloak.generateViewingKey(
        body.scope,
        body.label,
        expiresAt
      );

      const record: ViewingKeyRecord = {
        id: uuidv4(),
        key: rawKey,
        scope: body.scope,
        label: body.label,
        createdAt: new Date().toISOString(),
        expiresAt: body.expiresAt,
        findingIds: body.findingIds,
      };

      store.saveViewingKey(record);

      // Return record without the raw key — caller fetches via /raw when needed
      const { key: _k, ...safe } = record;
      return res.status(201).json(safe);
    } catch (err: any) {
      return res.status(500).json({ error: err.message });
    }
  });

  router.get("/", (_req, res) => {
    const keys = store.listViewingKeys().map(({ key: _k, ...safe }) => safe);
    return res.json(keys);
  });

  router.get("/:id", (req, res) => {
    const record = store.getViewingKey(req.params.id);
    if (!record) return res.status(404).json({ error: "Key not found" });
    const { key: _k, ...safe } = record;
    return res.json(safe);
  });

  router.get("/:id/raw", (req, res) => {
    const record = store.getViewingKey(req.params.id);
    if (!record) return res.status(404).json({ error: "Key not found" });

    if (record.expiresAt && new Date(record.expiresAt) < new Date()) {
      return res.status(410).json({ error: "Viewing key has expired" });
    }

    return res.json({ id: record.id, key: record.key, scope: record.scope });
  });

  router.delete("/:id", (req, res) => {
    const deleted = store.deleteViewingKey(req.params.id);
    if (!deleted) return res.status(404).json({ error: "Key not found" });
    return res.json({ deleted: true });
  });

  return router;
}
