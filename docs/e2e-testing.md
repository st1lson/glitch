# E2E Testing Integration (Cypress & Playwright)

While Glitch is fantastic for local development, it shines even brighter when integrated directly into your CI/CD pipelines alongside E2E test runners like Cypress or Playwright.

By utilizing Glitch's **Internal Control API (`/_glitch`)**, you can dynamically toggle network chaos, pause requests, and assert loading states *deterministically* inside your tests without mocking backend responses or writing complex interceptors.

## The `X-Glitch-Scenario` Header

To ensure that parallel test suites don't step on each other's toes, Glitch uses **Scenario Isolation**. 
When interacting with the Control API, always pass the `X-Glitch-Scenario` HTTP header with a unique scenario ID (e.g., your test name). Glitch will only apply those chaos rules to requests that also contain that header.

*If you don't pass this header, the changes apply to the "default" global scenario, which is fine for sequential testing but problematic for parallel testing.*

---

## 1. Playwright Integration

### Setting up the Proxy
Run your Playwright tests against your application, and ensure your application directs its API calls through Glitch.

### Example: Testing a 500 Error Recovery

You want to assert that when the backend throws a 500 error, your app shows a specific error toast.

```javascript
import { test, expect } from '@playwright/test';

test('Shows error toast on 500 Internal Server Error', async ({ page, request }) => {
  const scenario = 'test-500-error';

  // 1. Tell Glitch to inject a 100% failure rate with a 500 status code
  // We use PATCH to merge these rules on top of our baseline config
  await request.patch('http://localhost:3000/_glitch/rules', {
    headers: { 'X-Glitch-Scenario': scenario },
    data: {
      failure: {
        rate: 100,
        statuses: [{ code: 500, rate: 100 }]
      }
    }
  });

  // 2. Set the scenario header on the page so Glitch knows to apply the rules
  await page.setExtraHTTPHeaders({ 'X-Glitch-Scenario': scenario });

  // 3. Trigger the action in the UI
  await page.goto('/dashboard');
  await page.click('#submit-btn');

  // 4. Assert the error boundary or toast notification appears
  await expect(page.locator('.toast-error')).toBeVisible();

  // 5. Clean up the rules to reset this scenario
  await request.delete('http://localhost:3000/_glitch/rules', {
    headers: { 'X-Glitch-Scenario': scenario }
  });
});
```

### Example: Testing a Loading Spinner (Pause & Resume)

Asserting loading spinners can be notoriously flaky in E2E tests because the API might respond too quickly. With Glitch, you can permanently pause the request, assert the spinner, and then release it.

```javascript
test('Shows loading spinner while fetching data', async ({ page, request }) => {
  const scenario = 'test-spinner';

  // 1. Tell Glitch to pause the next request indefinitely (or add ?timeout=10s as a failsafe)
  await request.post(`http://localhost:3000/_glitch/pause?timeout=10s`, {
    headers: { 'X-Glitch-Scenario': scenario }
  });

  await page.setExtraHTTPHeaders({ 'X-Glitch-Scenario': scenario });

  // 2. Trigger the fetch
  await page.goto('/dashboard');
  await page.click('#load-data-btn');

  // 3. Assert the spinner is visible (the request is currently hanging in Glitch!)
  await expect(page.locator('.spinner')).toBeVisible();

  // 4. Tell Glitch to resume the request
  await request.post('http://localhost:3000/_glitch/resume', {
    headers: { 'X-Glitch-Scenario': scenario }
  });

  // 5. Assert the data eventually loaded
  await expect(page.locator('.data-table')).toBeVisible();
});
```

---

## 2. Cypress Integration

### Example: Testing a 500 Error Recovery

In Cypress, you can create a custom command to interact with the Glitch API, or just use `cy.request()`.

```javascript
describe('Error Handling', () => {
  const scenario = 'cypress-error-test';

  beforeEach(() => {
    // Intercept outbound requests from the browser to inject our scenario header
    cy.intercept('**/*', (req) => {
      req.headers['X-Glitch-Scenario'] = scenario;
    });
  });

  afterEach(() => {
    // Clean up Glitch rules after the test
    cy.request({
      method: 'DELETE',
      url: 'http://localhost:3000/_glitch/rules',
      headers: { 'X-Glitch-Scenario': scenario }
    });
  });

  it('displays a toast notification when the API fails', () => {
    // 1. Tell Glitch to throw 500s
    cy.request({
      method: 'PATCH',
      url: 'http://localhost:3000/_glitch/rules',
      headers: { 'X-Glitch-Scenario': scenario },
      body: {
        failure: {
          rate: 100,
          statuses: [{ code: 500, rate: 100 }]
        }
      }
    });

    // 2. Trigger the UI
    cy.visit('/dashboard');
    cy.get('#submit-btn').click();

    // 3. Assert the toast
    cy.get('.toast-error').should('be.visible').and('contain', 'Internal Server Error');
  });
});
```

### Example: Testing a Loading Spinner (Pause & Resume)

```javascript
describe('Loading States', () => {
  const scenario = 'cypress-loading-test';

  beforeEach(() => {
    cy.intercept('**/*', (req) => {
      req.headers['X-Glitch-Scenario'] = scenario;
    });
  });

  it('displays a spinner while data is loading', () => {
    // 1. Pause the request
    cy.request({
      method: 'POST',
      url: 'http://localhost:3000/_glitch/pause?timeout=10s',
      headers: { 'X-Glitch-Scenario': scenario }
    });

    cy.visit('/dashboard');
    cy.get('#load-data-btn').click();

    // 2. Assert the spinner (request is hanging)
    cy.get('.spinner').should('be.visible');

    // 3. Resume the request
    cy.request({
      method: 'POST',
      url: 'http://localhost:3000/_glitch/resume',
      headers: { 'X-Glitch-Scenario': scenario }
    });

    // 4. Assert completion
    cy.get('.data-table').should('be.visible');
    cy.get('.spinner').should('not.exist');
  });
});
```

## Available Endpoints

- `GET /_glitch/health` - Check if Glitch is running and if the current scenario is paused.
- `GET /_glitch/config` - Get the current merged configuration for the scenario.
- `GET /_glitch/config/baseline` - Get the baseline configuration.
- `PATCH /_glitch/rules` - Merge/overlay chaos rules onto the scenario.
- `DELETE /_glitch/rules` - Reset the scenario to the baseline configuration.
- `POST /_glitch/profile/{name}` - Apply a predefined chaos profile to the scenario.
- `POST /_glitch/pause?timeout=5s` - Pause all requests for this scenario.
- `POST /_glitch/resume` - Resume all requests for this scenario.
