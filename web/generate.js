#!/usr/bin/env node

/**
 * @param {string} input Source OpenAPI file
 * @param {string} output Output d.ts file
 * @return {Promise<void>}
 */
// const generate = async (input, output) => {
//   const ast = await openapiTS(new URL(input, import.meta.url), {
//     arrayLength: true,
//     enum: true,
//     additionalProperties: false,
//     // https://openapi-ts.pages.dev/node#example-blob-types
//     transform: (schemaObject, options) => {
//       if (schemaObject.format === 'binary') {
//         return schemaObject.nullable
//           ? ts.factory.createUnionTypeNode([BLOB, NULL])
//           : BLOB;
//       }
//     },
//   })
//
//   fs.writeFileSync(output, astToString(ast))
// }
