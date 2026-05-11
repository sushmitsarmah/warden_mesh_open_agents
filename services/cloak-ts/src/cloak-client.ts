/**
 * Cloak SDK wrapper.
 *
 * Tries to load the real @cloak-network/sdk at runtime.
 * Falls back to the local stub implementation so the service runs in
 * development/CI without the SDK package installed.
 *
 * To wire in the real SDK:
 *   npm install @cloak-network/sdk
 *   Update the dynamic import path below to "@cloak-network/sdk".
 */

import {
  AuditHistoryResult,
  Mint,
  PayoutRecipient,
  TransactionRecord,
  ViewingKeyRecord,
  ViewingKeyScope,
} from "./types.js";
import { ICloakSDK } from "./sdk-types.js";
import { StubCloakSDK } from "./sdk-stub.js";

async function loadSDK(_rpcUrl: string, _privateKey: string): Promise<ICloakSDK> {
  // ── Real SDK swap ──────────────────────────────────────────────────────────
  // When @cloak-network/sdk is available:
  //   1. yarn add @cloak-network/sdk @solana/web3.js bs58
  //   2. Replace this function body with:
  //
  //   const { CloakClient: SDK } = require("@cloak-network/sdk");
  //   const { Connection, Keypair } = require("@solana/web3.js");
  //   const bs58 = require("bs58");
  //   const connection = new Connection(_rpcUrl, "confirmed");
  //   const keypair = Keypair.fromSecretKey(bs58.decode(_privateKey));
  //   return new SDK({ connection, keypair });
  // ──────────────────────────────────────────────────────────────────────────
  console.warn("[cloak] @cloak-network/sdk not installed — using stub SDK");
  return new StubCloakSDK();
}

export class CloakClient {
  private sdk!: ICloakSDK;
  private ready = false;
  private rpcUrl: string;
  private privateKey: string;

  constructor(rpcUrl: string, privateKey: string) {
    this.rpcUrl = rpcUrl;
    this.privateKey = privateKey;
  }

  async init(): Promise<void> {
    this.sdk = await loadSDK(this.rpcUrl, this.privateKey);
    await this.sdk.init();
    this.ready = true;
  }

  async generateStealthAddress(): Promise<string> {
    this.assertReady();
    try {
      const addr = await this.sdk.generateStealthAddress();
      return addr.toString();
    } catch (err) {
      throw new Error(`Cloak generateStealthAddress failed: ${err}`);
    }
  }

  async batchDisbursement(recipients: PayoutRecipient[], mint: Mint): Promise<string> {
    this.assertReady();
    try {
      const sdkRecipients = recipients.map((r) => ({
        stealthAddress: r.stealthAddress,
        amount: BigInt(r.amountUsdc),
        mint,
      }));
      return await this.sdk.batchDisbursement({ recipients: sdkRecipients });
    } catch (err) {
      throw new Error(`Cloak batchDisbursement failed: ${err}`);
    }
  }

  async generateViewingKey(
    scope: ViewingKeyScope,
    _label: string,
    expiresAt?: Date
  ): Promise<{ rawKey: string }> {
    this.assertReady();
    try {
      const sdkScope = scope === "amounts_only" ? "amounts" : "full";
      const vk = await this.sdk.generateViewingKey({
        scope: sdkScope,
        ...(expiresAt ? { expiresAt } : {}),
      });
      return { rawKey: vk.toString() };
    } catch (err) {
      throw new Error(`Cloak generateViewingKey failed: ${err}`);
    }
  }

  async getHistory(rawViewingKey: string): Promise<TransactionRecord[]> {
    this.assertReady();
    try {
      const vk = await this.sdk.parseViewingKey(rawViewingKey);
      const entries = await this.sdk.getHistory(vk);
      return entries.map((e) => ({
        id: e.id,
        type: e.type as TransactionRecord["type"],
        mint: e.mint as Mint,
        amountUsdc: Number(e.amount),
        txSignature: e.signature,
        timestamp: e.timestamp.toISOString(),
        ...(e.sender ? { senderShielded: e.sender.toString() } : {}),
        ...(e.recipient ? { recipientShielded: e.recipient.toString() } : {}),
      }));
    } catch (err) {
      throw new Error(`Cloak getHistory failed: ${err}`);
    }
  }

  async buildAuditHistory(record: ViewingKeyRecord): Promise<AuditHistoryResult> {
    const now = new Date();
    if (record.expiresAt && new Date(record.expiresAt) < now) {
      throw new Error(`Viewing key "${record.id}" has expired`);
    }

    const transactions = await this.getHistory(record.key);
    const filtered =
      record.findingIds.length > 0
        ? transactions.filter(
            (t) => !t.findingId || record.findingIds.includes(t.findingId)
          )
        : transactions;

    return {
      viewingKeyId: record.id,
      scope: record.scope,
      label: record.label,
      expiresAt: record.expiresAt,
      transactions: filtered,
      totalAmountUsdc: filtered.reduce((s, t) => s + t.amountUsdc, 0),
      generatedAt: now.toISOString(),
    };
  }

  private assertReady(): void {
    if (!this.ready) {
      throw new Error("CloakClient not initialized — call init() first");
    }
  }
}
