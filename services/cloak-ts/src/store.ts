/**
 * In-memory store for viewing key records and payout history.
 * In production replace with a database — the interface is intentionally thin.
 */

import { PayoutResult, ViewingKeyRecord } from "./types.js";

class Store {
  private viewingKeys = new Map<string, ViewingKeyRecord>();
  private payouts = new Map<string, PayoutResult>();

  // ── Viewing keys ──────────────────────────────────────────────────────────

  saveViewingKey(record: ViewingKeyRecord): void {
    this.viewingKeys.set(record.id, record);
  }

  getViewingKey(id: string): ViewingKeyRecord | undefined {
    return this.viewingKeys.get(id);
  }

  listViewingKeys(): ViewingKeyRecord[] {
    return Array.from(this.viewingKeys.values());
  }

  deleteViewingKey(id: string): boolean {
    return this.viewingKeys.delete(id);
  }

  // ── Payouts ───────────────────────────────────────────────────────────────

  savePayout(result: PayoutResult): void {
    this.payouts.set(result.findingId, result);
  }

  getPayout(findingId: string): PayoutResult | undefined {
    return this.payouts.get(findingId);
  }

  listPayouts(): PayoutResult[] {
    return Array.from(this.payouts.values());
  }
}

export const store = new Store();
