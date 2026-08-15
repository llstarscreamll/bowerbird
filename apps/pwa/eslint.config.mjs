// @ts-check
import tseslint from 'typescript-eslint';
import angular from 'angular-eslint';

/**
 * PWA ESLint config.
 *
 * Page components use the Angular type-suffix convention:
 *   *.page.ts / *.page.html / *.page.css  (e.g. master.page.html)
 * with class names ending in `Page`.
 *
 * Regular UI components keep the `*.component.*` + `Component` suffix.
 *
 * Generate a new page with:
 *   ng g c path/to/name --type=page
 */
export default tseslint.config(
  {
    ignores: ['dist/**', '.angular/**', 'node_modules/**', 'src/app/shared/ui/**'],
  },
  {
    files: ['**/*.ts'],
    extends: [...angular.configs.tsRecommended],
    processor: angular.processInlineTemplates,
    rules: {
      // Allow both routed pages (`LoginPage`) and reusable components (`FileUploadComponent`).
      '@angular-eslint/component-class-suffix': [
        'error',
        {
          suffixes: ['Component', 'Page'],
        },
      ],
    },
  },
  {
    files: ['**/*.html', '**/*.page.html'],
    extends: [...angular.configs.templateRecommended],
    rules: {},
  },
);
