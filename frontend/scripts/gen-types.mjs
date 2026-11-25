import { convert } from "swagger2openapi";
import { writeFileSync, readFileSync } from "fs";
import { execSync } from "child_process";
import { join } from "path";

// The Makefile runs: cd frontend && node scripts/gen-types.mjs
// So process.cwd() is the frontend directory.
const frontendDir = process.cwd();
const root = join(frontendDir, "..");
const swaggerPath = join(root, "backend", "docs", "swagger.json");
const outputPath = join(frontendDir, "src", "lib", "generated", "api.ts");

console.log("Reading Swagger 2.0 spec from", swaggerPath);
let spec = JSON.parse(readFileSync(swaggerPath, "utf8"));

// Ensure required info fields for swagger2openapi conversion
spec.info = spec.info || {};
spec.info.title = spec.info.title || "Somo Tracker API";
spec.info.version = spec.info.version || "1.0.0";

console.log("Converting Swagger 2.0 → OpenAPI 3.0...");
convert(spec, {}, (err, result) => {
  if (err) {
    console.error("Conversion failed:", err);
    process.exit(1);
  }

  const openapiDoc = result.openapi;
  const tmpPath = "/tmp/swagger-openapi.json";
  writeFileSync(tmpPath, JSON.stringify(openapiDoc, null, 2));

  console.log("Generating TypeScript types...");
  try {
    execSync(`npx openapi-typescript ${tmpPath} -o ${outputPath}`, {
      stdio: "inherit",
      cwd: frontendDir,
    });
  } catch (e) {
    console.error("Type generation failed:", e);
    process.exit(1);
  }

  console.log("Types written to", outputPath);
});
