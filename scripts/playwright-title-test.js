import { browser } from 'k6/browser';
import { check } from 'https://jslib.k6.io/k6-utils/1.5.0/index.js';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 1,
      options: {
        browser: {
          type: 'chromium',
        },
      },
    },
  },
  thresholds: {
    checks: ['rate==1.0'],
    browser_web_vital_lcp: ['p(95)<2500'],
    browser_web_vital_fcp: ['p(95)<1800'],
    browser_web_vital_inp: ['p(95)<200'],
  },
};

export default async function () {
  const page = await browser.newPage();

  try {
    await page.goto('https://playwright.dev/');

    const title = await page.title();

    check(title, {
      'page title contains "Playwright"': (t) => t.includes('Playwright'),
    });
  } finally {
    await page.close();
  }
}
