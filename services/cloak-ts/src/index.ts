/**
 * Cloak Service — private payout and audit REST API
 *
 * Endpoints:
 *   POST /payout                    Batch-disburse shielded bug-bounty payments
 *   GET  /payout/:findingId         Get payout status
 *   GET  /payout                    List all payouts
 *
 *   POST /viewing-key               Issue a scoped viewing key
 *   GET  /viewing-key               List all keys (metadata only)
 *   GET  /viewing-key/:id           Get key metadata
 *   GET  /viewing-key/:id/raw       Return raw key string
 *   DELETE /viewing-key/:id         Revoke a key
 *
 *   GET  /audit/history?keyId=      Decrypt history (stored key)
 *   POST /audit/history             Decrypt history (raw key in body)
 *   GET  /audit/export?keyId=&fmt=  Export CSV or JSON
 *   GET  /audit/stealth             Generate researcher stealth address
 *
 *   GET  /health                    Health check
 *
 * Environment variables:
 *   CLOAK_RPC_URL          Solana RPC endpoint (default: mainnet-beta)
 *   CLOAK_PRIVATE_KEY      Base58 private key for the swarm's shielded wallet
 *   CLOAK_SERVICE_PORT     HTTP port (default: 4000)
 */

import express from "express";
import { CloakClient } from "./cloak-client.js";
import { payoutRouter } from "./routes/payout.js";
import { viewingKeyRouter } from "./routes/viewing-key.js";
import { auditRouter } from "./routes/audit.js";

const PORT = parseInt(process.env.CLOAK_SERVICE_PORT ?? "4000", 10);
const RPC_URL =
  process.env.CLOAK_RPC_URL ?? "https://api.mainnet-beta.solana.com";
const PRIVATE_KEY = process.env.CLOAK_PRIVATE_KEY ?? "";

async function main() {
  if (!PRIVATE_KEY) {
    console.error("CLOAK_PRIVATE_KEY not set — exiting");
    process.exit(1);
  }

  const cloak = new CloakClient(RPC_URL, PRIVATE_KEY);
  await cloak.init();
  console.log("Cloak SDK initialized");

  const app = express();
  app.use(express.json());

  app.use("/payout", payoutRouter(cloak));
  app.use("/viewing-key", viewingKeyRouter(cloak));
  app.use("/audit", auditRouter(cloak));

  app.get("/health", (_req, res) => {
    res.json({ status: "ok", service: "cloak-ts", rpc: RPC_URL });
  });

  app.listen(PORT, () => {
    console.log(`Cloak service listening on :${PORT}`);
  });
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
