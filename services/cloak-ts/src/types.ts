export type Mint = "USDC" | "USDT" | "SOL";

export type ViewingKeyScope =
  | "full"          // all fields: amount, sender, recipient, timestamp
  | "amounts_only"  // amounts + timestamps, no addresses
  | "time_limited"; // full but expires at a fixed wall-clock time

export interface ViewingKeyRecord {
  id: string;
  key: string;
  scope: ViewingKeyScope;
  label: string;         // human-readable: "Acme Inc audit Q2-2026"
  createdAt: string;     // ISO-8601
  expiresAt?: string;    // ISO-8601; absent → never expires
  findingIds: string[];  // the disclosures this key covers
}

export interface PayoutRecipient {
  stealthAddress: string;
  amountUsdc: number;   // in USDC base units (6 decimals)
  label?: string;       // optional memo for internal records only
}

export interface PayoutRequest {
  findingId: string;
  recipients: PayoutRecipient[];
  mint: Mint;
  generateViewingKey: boolean;
  viewingKeyLabel?: string;
  viewingKeyScope?: ViewingKeyScope;
  viewingKeyExpiresAt?: string;
}

export interface PayoutResult {
  txSignature: string;
  findingId: string;
  totalAmountUsdc: number;
  recipientCount: number;
  viewingKey?: ViewingKeyRecord;
  shieldedAt: string;
}

export interface TransactionRecord {
  id: string;
  findingId?: string;
  type: "disbursement" | "deposit" | "swap";
  mint: Mint;
  amountUsdc: number;
  recipientCount?: number;
  txSignature: string;
  timestamp: string;
  // only present when key scope includes addresses
  senderShielded?: string;
  recipientShielded?: string;
}

export interface AuditHistoryResult {
  viewingKeyId: string;
  scope: ViewingKeyScope;
  label: string;
  expiresAt?: string;
  transactions: TransactionRecord[];
  totalAmountUsdc: number;
  generatedAt: string;
}
