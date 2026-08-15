// @ts-check
import tseslint from 'typescript-eslint';
import angular from 'angular-eslint';

/**
 * PWA ESLint config.
 *
 * Page convention (per feature module):
 *   presentation/pages/<role>/<role>.page.ts|.html|.css
 *   e.g. master.page.ts, detail.page.ts, create.page.ts, edit.page.ts
 *   class suffix: Page
 *
 * Reusable UI keeps *.component.* + Component suffix.
 *
 * Generate a new page with:
 *   ng g c path/to/<role> --type=page
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
      // Pages: pages/<role>/<role>.page.ts with class <Role>Page (master, detail, login, …).
      // Components: *.component.ts with class *Component.
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
