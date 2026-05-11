"use client";

import { useState, useCallback } from "react";

// ── Types (mirror from cloak-ts/src/types.ts) ────────────────────────────────

type Scope = "full" | "amounts_only" | "time_limited";

interface Transaction {
  id: string;
  findingId?: string;
  type: string;
  mint: string;
  amountUsdc: number;
  txSignature: string;
  timestamp: string;
  senderShielded?: string;
  recipientShielded?: string;
}

interface AuditResult {
  viewingKeyId: string;
  scope: Scope;
  label: string;
  expiresAt?: string;
  transactions: Transaction[];
  totalAmountUsdc: number;
  generatedAt: string;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatUsdc(units: number): string {
  return `$${(units / 1_000_000).toFixed(2)}`;
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleString("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function scopeBadge(scope: Scope) {
  if (scope === "full") return <span className="badge-full">FULL AUDIT</span>;
  if (scope === "amounts_only") return <span className="badge-amounts">AMOUNTS ONLY</span>;
  return <span className="badge-time">TIME-LIMITED</span>;
}

function truncate(s: string, n = 12): string {
  if (!s) return "";
  return s.length <= n ? s : `${s.slice(0, 6)}…${s.slice(-6)}`;
}

// ── Main component ────────────────────────────────────────────────────────────

export default function AuditDashboard() {
  const [mode, setMode] = useState<"keyId" | "rawKey">("keyId");
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<AuditResult | null>(null);
  const [stealthAddr, setStealthAddr] = useState<string | null>(null);
  const [stealthLoading, setStealthLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  const fetchHistory = useCallback(async () => {
    if (!input.trim()) return;
    setLoading(true);
    setError(null);
    setResult(null);

    try {
      let res: Response;
      if (mode === "keyId") {
        res = await fetch(`/api/history?keyId=${encodeURIComponent(input.trim())}`);
      } else {
        res = await fetch("/api/history", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ key: input.trim(), label: "External audit" }),
        });
      }

      const data = await res.json();
      if (!res.ok) {
        setError(data.error ?? "Unknown error");
      } else {
        setResult(data as AuditResult);
      }
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [input, mode]);

  const exportCsv = useCallback(() => {
    if (!result) return;
    window.location.href = `/api/export?keyId=${encodeURIComponent(result.viewingKeyId)}&fmt=csv`;
  }, [result]);

  const exportJson = useCallback(() => {
    if (!result) return;
    window.location.href = `/api/export?keyId=${encodeURIComponent(result.viewingKeyId)}&fmt=json`;
  }, [result]);

  const generateStealth = useCallback(async () => {
    setStealthLoading(true);
    try {
      const res = await fetch("/api/stealth");
      const data = await res.json();
      setStealthAddr(data.stealthAddress);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setStealthLoading(false);
    }
  }, []);

  const copyToClipboard = useCallback((text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, []);

  return (
    <div className="space-y-10">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">
          Compliance Audit Dashboard
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          Input a viewing key to decrypt shielded bug-bounty payment history.
          No raw wallet addresses or amounts appear on-chain.
        </p>
      </div>

      {/* Input card */}
      <div className="rounded-xl border border-gray-800 bg-gray-900 p-6 space-y-4">
        <div className="flex gap-2">
          <button
            onClick={() => { setMode("keyId"); setInput(""); setResult(null); }}
            className={`px-3 py-1.5 rounded text-xs font-semibold transition-colors ${
              mode === "keyId"
                ? "bg-emerald-700 text-white"
                : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            }`}
          >
            Key ID (stored)
          </button>
          <button
            onClick={() => { setMode("rawKey"); setInput(""); setResult(null); }}
            className={`px-3 py-1.5 rounded text-xs font-semibold transition-colors ${
              mode === "rawKey"
                ? "bg-emerald-700 text-white"
                : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            }`}
          >
            Raw viewing key
          </button>
        </div>

        <div className="flex gap-3">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && fetchHistory()}
            placeholder={
              mode === "keyId"
                ? "e.g. 3fa85f64-5717-4562-b3fc-2c963f66afa6"
                : "Paste raw Cloak viewing key…"
            }
            className="flex-1 rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5
                       text-sm text-gray-200 placeholder-gray-600
                       focus:outline-none focus:border-emerald-500 focus:ring-1 focus:ring-emerald-500"
          />
          <button
            onClick={fetchHistory}
            disabled={loading || !input.trim()}
            className="px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-500
                       disabled:opacity-40 disabled:cursor-not-allowed
                       text-sm font-semibold text-white transition-colors"
          >
            {loading ? "Decrypting…" : "Decrypt"}
          </button>
        </div>

        {error && (
          <p className="text-sm text-red-400 bg-red-950 border border-red-800 rounded px-3 py-2">
            {error}
          </p>
        )}
      </div>

      {/* Results */}
      {result && (
        <div className="space-y-6">
          {/* Summary bar */}
          <div className="rounded-xl border border-gray-800 bg-gray-900 p-5
                          grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wider">Key scope</p>
              <p className="mt-1">{scopeBadge(result.scope)}</p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wider">Label</p>
              <p className="mt-1 text-sm text-gray-200 truncate">{result.label}</p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wider">Total disbursed</p>
              <p className="mt-1 text-lg font-bold text-emerald-400">
                {formatUsdc(result.totalAmountUsdc)}
              </p>
            </div>
            <div>
              <p className="text-xs text-gray-500 uppercase tracking-wider">Transactions</p>
              <p className="mt-1 text-lg font-bold text-white">{result.transactions.length}</p>
            </div>

            {result.expiresAt && (
              <div className="col-span-2 md:col-span-4 border-t border-gray-800 pt-3">
                <p className="text-xs text-amber-400">
                  Key expires: {fmtDate(result.expiresAt)}
                </p>
              </div>
            )}
          </div>

          {/* Export buttons */}
          <div className="flex gap-2 justify-end">
            <button
              onClick={exportCsv}
              className="px-4 py-2 rounded-lg bg-gray-800 hover:bg-gray-700
                         text-xs font-semibold text-gray-300 transition-colors"
            >
              Export CSV
            </button>
            <button
              onClick={exportJson}
              className="px-4 py-2 rounded-lg bg-gray-800 hover:bg-gray-700
                         text-xs font-semibold text-gray-300 transition-colors"
            >
              Export JSON
            </button>
          </div>

          {/* Transaction table */}
          <div className="rounded-xl border border-gray-800 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-900 border-b border-gray-800">
                <tr>
                  <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Timestamp</th>
                  <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Type</th>
                  <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Amount</th>
                  <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Finding</th>
                  {result.scope === "full" && (
                    <>
                      <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Sender</th>
                      <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Recipient</th>
                    </>
                  )}
                  <th className="px-4 py-3 text-left text-xs text-gray-500 uppercase tracking-wider">Tx</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {result.transactions.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-gray-600 text-sm">
                      No transactions found under this viewing key.
                    </td>
                  </tr>
                )}
                {result.transactions.map((tx) => (
                  <tr key={tx.id} className="bg-gray-950 hover:bg-gray-900 transition-colors">
                    <td className="px-4 py-3 text-gray-400 whitespace-nowrap">
                      {fmtDate(tx.timestamp)}
                    </td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs
                                       bg-gray-800 text-gray-300 font-mono">
                        {tx.type}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-emerald-400 font-semibold">
                      {formatUsdc(tx.amountUsdc)} {tx.mint}
                    </td>
                    <td className="px-4 py-3 text-gray-500 font-mono text-xs">
                      {tx.findingId ? truncate(tx.findingId, 16) : "—"}
                    </td>
                    {result.scope === "full" && (
                      <>
                        <td className="px-4 py-3 text-gray-500 font-mono text-xs">
                          {tx.senderShielded ? truncate(tx.senderShielded) : "—"}
                        </td>
                        <td className="px-4 py-3 text-gray-500 font-mono text-xs">
                          {tx.recipientShielded ? truncate(tx.recipientShielded) : "—"}
                        </td>
                      </>
                    )}
                    <td className="px-4 py-3">
                      <span
                        title={tx.txSignature}
                        className="font-mono text-xs text-sky-500 hover:text-sky-400 cursor-pointer"
                        onClick={() => copyToClipboard(tx.txSignature)}
                      >
                        {truncate(tx.txSignature)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <p className="text-xs text-gray-700 text-right">
            Generated {fmtDate(result.generatedAt)} · Key ID {result.viewingKeyId}
          </p>
        </div>
      )}

      {/* Stealth address generator */}
      <div className="rounded-xl border border-gray-800 bg-gray-900 p-6 space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
            Researcher Stealth Address
          </h2>
          <p className="mt-1 text-xs text-gray-600">
            Generate a one-time receive address for a security researcher.
            Their real wallet never appears on-chain — even to the payer.
          </p>
        </div>

        <button
          onClick={generateStealth}
          disabled={stealthLoading}
          className="px-5 py-2.5 rounded-lg bg-sky-700 hover:bg-sky-600
                     disabled:opacity-40 disabled:cursor-not-allowed
                     text-sm font-semibold text-white transition-colors"
        >
          {stealthLoading ? "Generating…" : "Generate stealth address"}
        </button>

        {stealthAddr && (
          <div
            className="flex items-center gap-3 bg-gray-800 border border-gray-700
                        rounded-lg px-4 py-3 cursor-pointer group"
            onClick={() => copyToClipboard(stealthAddr)}
          >
            <code className="flex-1 text-xs text-sky-300 break-all font-mono">
              {stealthAddr}
            </code>
            <span className="text-xs text-gray-600 group-hover:text-gray-400 transition-colors whitespace-nowrap">
              {copied ? "Copied!" : "Click to copy"}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
