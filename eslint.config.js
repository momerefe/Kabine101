import js from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['web/src/api/schema.gen.ts'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
);
