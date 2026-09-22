const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");
const vm = require("node:vm");
const ts = require("typescript");

class Component {

  constructor(props) {

    this.props = props;

  }

  setState(update) {

    this.state = { ...this.state, ...update };

  }

}

function loadComponent(file, dependencies = {}, globals = {}) {

  const exports = {};
  const timers = new Map();
  let nextTimer = 0;

  const source = readFileSync(path.join(__dirname, "../src/Features/Player", file), "utf8");
  const compiled = ts.transpileModule(source, {

    compilerOptions: {

      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.CommonJS,
      jsx: ts.JsxEmit.ReactJSX,

    },

  }).outputText;

  vm.runInNewContext(compiled, {

    exports,
    require(name) {

      if (name === "react") return { Component, createRef: () => ({ current: null }) };
      if (name === "react/jsx-runtime") return require(name);
      if (name === "@/Utils/ClassNames") return { cn: (...values) => values.filter(Boolean).join(" ") };
      if (name in dependencies) return dependencies[name];

      throw new Error(`Unexpected dependency: ${name}`);

    },
    setInterval: (callback) => {

      timers.set(++nextTimer, callback);
      return nextTimer;

    },
    clearInterval: (id) => timers.delete(id),
    ...globals,

  });

  return { ...exports, timers };

}

test("captions follow cue boundaries and backward seeks, and release their timer", async () => {

  const cues = [{ start: 1, end: 2, text: "First" }, { start: 3, end: 5, text: "Second" }];
  const { SubtitleDisplay, timers } = loadComponent("Subtitles/SubtitleDisplay.tsx", {

    "@/Features/Player/Subtitles/Vtt": { loadSubtitleCues: async () => cues },

  });
  const video = { currentTime: 1.5 };
  const display = new SubtitleDisplay({ videoRef: { current: video }, track: { id: "a", proxyUrl: "/a" } });

  await display.loadTrack(display.props.track);
  assert.equal(display.state.cue.text, "First");
  assert.equal(timers.size, 1);

  for (const [time, expected] of [[2, null], [3, "Second"], [5, null], [1, "First"]]) {

    video.currentTime = time;
    display.sync();
    assert.equal(display.state.cue?.text ?? null, expected);

  }

  await display.loadTrack(null);
  assert.equal(display.state.cue, null);
  assert.equal(timers.size, 0);

  await display.loadTrack(display.props.track);
  display.componentWillUnmount();
  assert.equal(timers.size, 0);

});

test("late subtitle downloads cannot replace a newer track or restart after unmount", async () => {

  const pending = new Map();
  const { SubtitleDisplay, timers } = loadComponent("Subtitles/SubtitleDisplay.tsx", {

    "@/Features/Player/Subtitles/Vtt": {

      loadSubtitleCues: (url) => new Promise((resolve) => pending.set(url, resolve)),

    },

  });
  const display = new SubtitleDisplay({ videoRef: { current: { currentTime: 1 } }, track: null });
  const oldLoad = display.loadTrack({ proxyUrl: "/old" });
  const newLoad = display.loadTrack({ proxyUrl: "/new" });

  pending.get("/new")([{ start: 0, end: 2, text: "New" }]);
  await newLoad;
  pending.get("/old")([{ start: 0, end: 2, text: "Old" }]);
  await oldLoad;
  assert.equal(display.state.cue.text, "New");

  const finalLoad = display.loadTrack({ proxyUrl: "/last" });
  display.componentWillUnmount();
  pending.get("/last")([{ start: 0, end: 2, text: "Late" }]);
  await finalLoad;
  assert.equal(timers.size, 0);

});

test("ambience settles without an endless animation loop and suspends hidden or paused work", () => {

  const frames = new Map();
  let nextFrame = 0;
  const document = {

    hidden: false,
    createElement: () => ({ getContext: () => null }),
    removeEventListener: () => {},

  };
  const { AmbienceLayer, timers } = loadComponent("AmbienceLayer.tsx", {}, {

    document,
    performance: { now: () => 10000 },
    requestAnimationFrame: (callback) => {

      frames.set(++nextFrame, callback);
      return nextFrame;

    },
    cancelAnimationFrame: (id) => frames.delete(id),

  });
  const video = { paused: true, readyState: 2, videoWidth: 1920 };
  const layer = new AmbienceLayer({ videoRef: { current: video }, enabled: true });
  let samples = 0;

  layer.extractTargets = () => samples++;
  layer.maybeSample();
  assert.equal(samples, 0);
  layer.maybeSample(true);
  assert.equal(samples, 1);

  layer.targetPrimary = { r: 100, g: 20, b: 60 };
  layer.startAnimation();

  for (let time = 40; frames.size && time < 20000; time += 40) {

    const [id, callback] = frames.entries().next().value;
    frames.delete(id);
    callback(time);

  }

  assert.equal(frames.size, 0);
  assert.equal(layer.displayPrimary.r, 100);

  layer.startSampling();
  layer.startAnimation();
  document.hidden = true;
  layer.onVisibilityChange();
  assert.equal(timers.size, 0);
  assert.equal(frames.size, 0);

  layer.maybeSample(true);
  assert.equal(samples, 1);
  layer.componentWillUnmount();

});
