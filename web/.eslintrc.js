/* global module */

module.exports = {
  root: true,
  parser: "vue-eslint-parser",
  parserOptions: {
    parser: "@typescript-eslint/parser",
  },
  extends: [
    "plugin:vue/strongly-recommended",
    "eslint:recommended",
    "@vue/typescript/recommended",
  ],
  plugins: ["@typescript-eslint"],
  rules: {},
  ignorePatterns: ["schema.gen.ts"],
}
