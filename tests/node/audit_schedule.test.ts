import test from "node:test";
import assert from "node:assert/strict";
import { runCli } from "../helpers/runner.ts";

test("audit projects the next settlement tranche without mutating the report", () => {
  const file = "tests/fixtures/partial_settlement.json";
  const before = JSON.parse(runCli(["run", file, "--json"]));
  const audit = JSON.parse(runCli(["audit", file, "--json"]));
  const after = JSON.parse(runCli(["run", file, "--json"]));
  assert.deepEqual(after, before);
  assert.equal(audit.windows.length, 1);
  assert.equal(audit.schedule.length, 1);
  assert.equal(audit.schedule[0].paymentId, "pay-2");
  assert.equal(audit.schedule[0].outstanding, 700);
  assert.equal(audit.schedule[0].trancheLimit, 700);
  assert.equal(audit.schedule[0].feeEstimate, 21);
  assert.equal(audit.schedule[0].netEstimate, 679);
});

test("audit text mode presents window and payment projection", () => {
  const output = runCli(["audit", "tests/fixtures/partial_settlement.json"]);
  assert.match(output, /window=noon-1 asset=usd/);
  assert.match(output, /payment=pay-2 due=3 tranche=700 fee=21 net=679/);
});
