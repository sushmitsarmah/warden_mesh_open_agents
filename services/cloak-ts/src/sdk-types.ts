/**
 * Interface contract for the Cloak SDK.
 *
 * When the real @cloak-network/sdk package is available:
 *   1. npm install @cloak-network/sdk
 *   2. In cloak-client.ts replace the stub import with:
 *        import { CloakSDK } from "@cloak-network/sdk";
 *        export type { ICloakSDK } from "./sdk-types.js";
 *   3. Verify the SDK's exported class satisfies ICloakSDK.
 */

export interface StealthAddress {
  toString(): string;
}

export type SDKScope = "full" | "amounts";

export interface SDKViewingKeyOpts {
  scope: SDKScope;
  expiresAt?: Date;
}

export interface SDKViewingKey {
  toString(): string;
}

export interface SDKHistoryEntry {
  id: string;
  type: string;
  mint: string;
  amount: bigint;
  signature: string;
  timestamp: Date;
  sender?: { toString(): string };
  recipient?: { toString(): string };
}

export interface SDKRecipient {
  stealthAddress: string;
  amount: bigint;
  mint: string;
}

export interface ICloakSDK {
  init(): Promise<void>;
  generateStealthAddress(): Promise<StealthAddress>;
  batchDisbursement(opts: { recipients: SDKRecipient[] }): Promise<string>;
  generateViewingKey(opts: SDKViewingKeyOpts): Promise<SDKViewingKey>;
  parseViewingKey(raw: string): Promise<SDKViewingKey>;
  getHistory(vk: SDKViewingKey): Promise<SDKHistoryEntry[]>;
}
