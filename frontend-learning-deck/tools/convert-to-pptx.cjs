const fs = require("fs/promises");
const path = require("path");
const os = require("os");
const Module = require("module");

const bundledModules = "C:\\Users\\MEIYING\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\node\\node_modules";
const bundledPnpmModules = path.join(bundledModules, ".pnpm", "node_modules");
process.env.NODE_PATH = [bundledPnpmModules, bundledModules, process.env.NODE_PATH || ""].filter(Boolean).join(path.delimiter);
Module._initPaths();

const pptxgen = require("pptxgenjs");
const html2pptx = require(path.join(__dirname, "html2pptx-local.js"));

async function main() {
  const deckRoot = path.resolve(__dirname, "..");
  const slidesDir = path.join(deckRoot, "slides");
  const outFile = path.resolve(deckRoot, "..", "frontend-learning-insights.pptx");
  const tmpDir = path.join(os.tmpdir(), "frontend-learning-html2pptx");
  await fs.mkdir(tmpDir, { recursive: true });

  const files = (await fs.readdir(slidesDir)).filter((file) => file.endsWith(".html")).sort();
  const pres = new pptxgen();
  pres.layout = "LAYOUT_WIDE";
  pres.author = "Codex";
  pres.subject = "学习前端的心得体会，以及对前端学习方法的见解";
  pres.title = "学习前端的心得体会";
  pres.company = "NovelOS";
  pres.lang = "zh-CN";
  pres.theme = {
    headFontFace: "Microsoft YaHei UI",
    bodyFontFace: "Microsoft YaHei UI",
    lang: "zh-CN"
  };

  const failures = [];
  for (const [index, file] of files.entries()) {
    const fullPath = path.join(slidesDir, file);
    try {
      await html2pptx(fullPath, pres, { tmpDir });
      console.log(`[${index + 1}/${files.length}] ${file} OK`);
    } catch (error) {
      failures.push({ file, error: error.message });
      console.error(`[${index + 1}/${files.length}] ${file} FAILED`);
      console.error(error.message);
    }
  }

  if (failures.length) {
    throw new Error(`${failures.length} slide(s) failed conversion.`);
  }

  await pres.writeFile({ fileName: outFile });
  console.log(`Wrote ${outFile}`);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
