import test from "node:test";
import assert from "node:assert/strict";
import { amount, byId, runFixture, validateFixture } from "../helpers/runner.ts";

test("deposits reserve liquidity and closing records the solar window snapshot", () => {
  const report = runFixture("deposit_close.json");
  const window = byId(report.windows, "dawn-1");
  const alice = byId(report.accounts, "alice");
  const payment = byId(report.payments, "pay-1");

  assert.equal(report.epoch, 2);
  assert.equal(window.status, "closed");
  assert.equal(window.reserve, 1000);
  assert.equal(window.outstanding, 600);
  assert.equal(window.snapshot?.reserve, 1000);
  assert.equal(window.snapshot?.outstanding, 600);
  assert.equal(window.snapshot?.solvencyMargin, 0);
  assert.equal(amount(alice.free, "usd"), 400);
  assert.equal(amount(alice.reserved, "usd"), 600);
  assert.equal(payment.status, "queued");
  assert.equal(validateFixture("deposit_close.json"), "ok");
});
