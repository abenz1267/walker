"use strict";
const os = require("os");

const args = process.argv.slice(2);
const term = args.join(" ");

const { spawnSync } = require("child_process");
const { stderr } = require("process");
const ls = spawnSync("rink", [term]);

if (ls.stderr.toString() !== "") {
  return [{}];
}

const res = ls.stdout.toString();
const lines = res.split(os.EOL);

console.log(
  JSON.stringify([
    {
      label: lines[1],
      sub: lines[0],
      searchable: term,
      class: "calc",
    },
  ]),
);
