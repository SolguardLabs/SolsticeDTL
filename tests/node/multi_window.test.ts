import test from "node:test";
import assert from "node:assert/strict";
import { byId, runFixture } from "../helpers/runner.ts";

test("independent solar windows keep reserves, snapshots and settlements separated", () => {
  const report = runFixture("multi_window.json");
  const dawn = byId(report.windows, "dawn-2");
  const noon = byId(report.windows, "noon-2");
  const pay4 = byId(report.payments, "pay-4");
  const pay5 = byId(report.payments, "pay-5");

  assert.equal(report.epoch, 3);
  assert.equal(dawn.status, "closed");
  assert.equal(noon.status, "closed");
  assert.equal(dawn.snapshot?.reserve, 1525);
  assert.equal(noon.snapshot?.reserve, 810);
  assert.equal(pay4.settled, 300);
  assert.equal(pay5.settled, 100);
  assert.equal(dawn.accruedFees, 8);
  assert.equal(noon.accruedFees, 3);
  assert.equal(report.metrics.settledGross, 400);
});
