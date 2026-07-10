import test from "node:test";
import assert from "node:assert/strict";
import { amount, byId, runFixture } from "../helpers/runner.ts";

test("post-close withdrawal consumes account free balance and records release metrics", () => {
  const report = runFixture("withdrawal_cycle.json");
  const window = byId(report.windows, "dusk-1");
  const alice = byId(report.accounts, "alice");
  const payment = byId(report.payments, "pay-3");

  assert.equal(window.status, "closed");
  assert.equal(window.snapshot?.reserve, 1200);
  assert.equal(window.snapshot?.outstanding, 800);
  assert.equal(window.snapshot?.accountFree.find((entry) => entry.id === "alice")?.amount, 400);
  assert.equal(window.reserve, 894);
  assert.equal(window.grossWithdrawals, 100);
  assert.equal(amount(window.postCloseWithdrawn, "alice"), 100);
  assert.equal(amount(alice.free, "usd"), 300);
  assert.equal(amount(alice.withdrawn, "usd"), 100);
  assert.equal(payment.outstanding, 600);
});
