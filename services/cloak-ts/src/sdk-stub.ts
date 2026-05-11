/**
 * Stub implementation of ICloakSDK for development and CI.
 *
 * All operations succeed but are no-ops — no real Solana transactions are sent.
 * Replace with the real SDK import in cloak-client.ts once the package is available.
 */

import crypto from "crypto";
import {
  ICloakSDK,
  SDKHistoryEntry,
  SDKRecipient,
  SDKViewingKey,
  SDKViewingKeyOpts,
  StealthAddress,
} from "./sdk-types.js";

class StubViewingKey implements SDKViewingKey {
  constructor(private raw: string) {}
  toString() { return this.raw; }
}

class StubStealthAddress implements StealthAddress {
  constructor(private addr: string) {}
  toString() { return this.addr; }
}

// In-memory store so getHistory returns what was disbursed in the same session
const disbursements: SDKHistoryEntry[] = [];
const viewingKeys = new Map<string, StubViewingKey>();

export class StubCloakSDK implements ICloakSDK {
  async init(): Promise<void> {
    console.warn("[cloak-stub] Using stub SDK — no real Solana transactions");
  }

  async generateStealthAddress(): Promise<StealthAddress> {
    const addr = "stealth" + crypto.randomBytes(16).toString("hex");
    return new StubStealthAddress(addr);
  }

  async batchDisbursement(opts: { recipients: SDKRecipient[] }): Promise<string> {
    const sig = "stubsig" + crypto.randomBytes(32).toString("hex");
    for (const r of opts.recipients) {
      disbursements.push({
        id: crypto.randomUUID(),
        type: "disbursement",
        mint: r.mint,
        amount: r.amount,
        signature: sig,
        timestamp: new Date(),
        recipient: { toString: () => r.stealthAddress },
      });
    }
    return sig;
  }

  async generateViewingKey(opts: SDKViewingKeyOpts): Promise<SDKViewingKey> {
    const raw = "vk_stub_" + crypto.randomBytes(20).toString("hex");
    const vk = new StubViewingKey(raw);
    viewingKeys.set(raw, vk);
    return vk;
  }

  async parseViewingKey(raw: string): Promise<SDKViewingKey> {
    return viewingKeys.get(raw) ?? new StubViewingKey(raw);
  }

  async getHistory(_vk: SDKViewingKey): Promise<SDKHistoryEntry[]> {
    return [...disbursements];
  }
}
