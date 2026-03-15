import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));

export const typeDefs = readFileSync(join(root, "schema.graphql"), "utf8");
export const operations = readFileSync(join(root, "operations.graphql"), "utf8");
