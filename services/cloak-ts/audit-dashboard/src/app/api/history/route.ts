/**
 * Next.js API route — proxy to Cloak service /audit/history
 *
 * Accepts both GET (?keyId=) and POST (raw key in body).
 * Keeps the Cloak service URL server-side so it never reaches the browser.
 */

export const dynamic = "force-dynamic";

import { NextRequest, NextResponse } from "next/server";

const SERVICE = process.env.NEXT_PUBLIC_CLOAK_SERVICE_URL ?? "http://localhost:4000";

export async function GET(req: NextRequest) {
  const keyId = req.nextUrl.searchParams.get("keyId");
  if (!keyId) {
    return NextResponse.json({ error: "keyId required" }, { status: 400 });
  }

  const upstream = await fetch(`${SERVICE}/audit/history?keyId=${encodeURIComponent(keyId)}`);
  const data = await upstream.json();
  return NextResponse.json(data, { status: upstream.status });
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const upstream = await fetch(`${SERVICE}/audit/history`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await upstream.json();
  return NextResponse.json(data, { status: upstream.status });
}
