export const dynamic = "force-dynamic";

import { NextResponse } from "next/server";

const SERVICE = process.env.NEXT_PUBLIC_CLOAK_SERVICE_URL ?? "http://localhost:4000";

export async function GET() {
  const upstream = await fetch(`${SERVICE}/audit/stealth`);
  const data = await upstream.json();
  return NextResponse.json(data, { status: upstream.status });
}
