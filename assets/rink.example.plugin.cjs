"use strict";
const os = require("os");

const args = process.argv.slice(2);
const term = args.join(" ");

if (term.length < 3) {
  console.log([]);
  return;
}

const { spawnSync } = require("child_process");
const { stderr } = require("process");
const ls = spawnSync("rink", [term]);

if (ls.stderr.toString() !== "") {
  return [{}];
}

const res = ls.stdout.toString();
const lines = res.split(os.EOL);

if (lines[1].includes("No such unit")) {
  console.log([]);
  return;
}

if (lines[1].includes("Expected")) {
  console.log([]);
  return;
}

console.log(
  JSON.stringify([
    {
      label: lines[1],
      sub: "rink",
      exec: `echo '${lines[1]}' | wl-copy`,
      class: "calc",
      matching: 1,
    },
  ]),
);
