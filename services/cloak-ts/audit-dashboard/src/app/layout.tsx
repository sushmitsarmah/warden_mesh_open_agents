import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Cloak Audit Dashboard — Warden Mesh",
  description: "Private bug-bounty compliance reporting via Cloak viewing keys",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-950 text-gray-100 min-h-screen font-mono">
        <nav className="border-b border-gray-800 px-6 py-4 flex items-center gap-3">
          <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          <span className="text-sm font-semibold tracking-widest text-gray-300 uppercase">
            Warden Mesh / Cloak Audit
          </span>
          <span className="ml-auto text-xs text-gray-600">
            Shielded payouts · Zero public trail
          </span>
        </nav>
        <main className="max-w-6xl mx-auto px-6 py-10">{children}</main>
      </body>
    </html>
  );
}
