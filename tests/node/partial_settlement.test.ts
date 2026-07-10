import test from "node:test";
import assert from "node:assert/strict";
import { amount, byId, runFixture } from "../helpers/runner.ts";

test("partial settlement applies service fees and leaves the remaining obligation queued", () => {
  const report = runFixture("partial_settlement.json");
  const window = byId(report.windows, "noon-1");
  const alice = byId(report.accounts, "alice");
  const merchant = byId(report.accounts, "merchant");
  const payment = byId(report.payments, "pay-2");

  assert.equal(payment.status, "settling");
  assert.equal(payment.settled, 200);
  assert.equal(payment.outstanding, 700);
  assert.equal(payment.fees, 6);
  assert.equal(window.settlementVolume, 200);
  assert.equal(window.accruedFees, 6);
  assert.equal(window.reserve, 794);
  assert.equal(amount(alice.reserved, "usd"), 700);
  assert.equal(amount(merchant.settledIn, "usd"), 194);
  assert.equal(report.metrics.settledGross, 200);
});
