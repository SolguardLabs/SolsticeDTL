import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export type AmountEntry = {
  id: string;
  amount: number;
};

export type AccountReport = {
  id: string;
  role: string;
  free: AmountEntry[];
  reserved: AmountEntry[];
  settledIn: AmountEntry[];
  withdrawn: AmountEntry[];
  feesPaid: AmountEntry[];
};

export type WindowReport = {
  id: string;
  asset: string;
  status: string;
  reserve: number;
  grossDeposits: number;
  grossWithdrawals: number;
  settlementVolume: number;
  accruedFees: number;
  outstanding: number;
  paymentCount: number;
  postCloseWithdrawn: AmountEntry[];
  snapshot?: {
    epoch: number;
    reserve: number;
    outstanding: number;
    accruedFees: number;
    settlementVolume: number;
    solvencyMargin: number;
    accountFree: AmountEntry[];
    accountReserved: AmountEntry[];
  };
};

export type PaymentReport = {
  id: string;
  windowId: string;
  source: string;
  beneficiary: string;
  asset: string;
  amount: number;
  settled: number;
  outstanding: number;
  fees: number;
  status: string;
};

export type SolsticeReport = {
  name: string;
  epoch: number;
  accounts: AccountReport[];
  windows: WindowReport[];
  payments: PaymentReport[];
  reconciliation: Array<Record<string, string | number>>;
  metrics: Record<string, number>;
};

export const root = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const binary = join(root, "bin", process.platform === "win32" ? "solsticedtl.exe" : "solsticedtl");

export function ensureBuilt(): void {
  if (existsSync(binary)) {
    return;
  }
  const result = spawnSync(process.execPath, ["scripts/build.mjs"], {
    cwd: root,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || "build failed");
  }
}

export function runCli(args: string[]): string {
  ensureBuilt();
  const result = spawnSync(binary, args, {
    cwd: root,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(`command failed: ${binary} ${args.join(" ")}\n${result.stderr}`);
  }
  return result.stdout;
}

export function runFixture(name: string): SolsticeReport {
  return JSON.parse(runCli(["run", join("tests", "fixtures", name), "--json"])) as SolsticeReport;
}

export function validateFixture(name: string): string {
  return runCli(["validate", join("tests", "fixtures", name)]).trim();
}

export function byId<T extends { id: string }>(items: T[], id: string): T {
  const found = items.find((item) => item.id === id);
  if (!found) {
    throw new Error(`missing ${id}`);
  }
  return found;
}

export function amount(entries: AmountEntry[], id: string): number {
  return entries.find((entry) => entry.id === id)?.amount ?? 0;
}
