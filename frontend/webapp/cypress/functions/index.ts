export * from './cy-alias';
import { DATA_IDS } from '../constants';

export const visitPage = (path: string, callback?: () => void) => {
  cy.visit(path);

  // Wait for the page to load.
  // On rare occasions, the page might still be blank, and cypress would attempt to interact with it, triggering false errors...
  cy.wait(1000).then(() => {
    if (!!callback) callback();
  });
};

interface GetCrdIdsOptions {
  namespace: string;
  crdName: string;
  expectedError: string;
  expectedLength: number;
}

export const getCrdIds = ({ namespace, crdName, expectedError, expectedLength }: GetCrdIdsOptions, callback?: (crdIds: string[]) => void) => {
  cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then(({ stderr, stdout }) => {
    expect(stderr).to.eq(expectedError);

    if (!!expectedError) {
      expect(stdout).to.eq('');
    } else {
      expect(stdout).to.not.be.empty;
    }

    const crdIds = stdout.split('\n').filter((str) => !!str);
    expect(crdIds.length).to.eq(expectedLength);

    if (!!callback) callback(crdIds);
  });
};

interface GetCrdByIdOptions {
  namespace: string;
  crdName: string;
  crdId: string;
  expectedError: string;
  expectedKey: string;
  expectedValue: string | boolean;
}

export const getCrdById = ({ namespace, crdName, crdId, expectedError, expectedKey, expectedValue }: GetCrdByIdOptions, callback?: () => void) => {
  cy.exec(`kubectl get ${crdName} ${crdId} -n ${namespace} -o json`).then(({ stderr, stdout }) => {
    expect(stderr).to.eq(expectedError);

    if (!!expectedError) {
      expect(stdout).to.eq('');
    } else {
      expect(stdout).to.not.be.empty;
    }

    const parsed = JSON.parse(stdout);
    const { spec } = parsed?.items?.[0] || parsed || {};

    expect(spec).to.not.be.empty;
    expect(spec[expectedKey]).to.eq(expectedValue);

    if (!!callback) callback();
  });
};

interface UpdateEntityOptions {
  nodeId: string;
  nodeContains?: string;
  fieldKey: string;
  fieldValue: string;
}

export const updateEntity = ({ nodeId, nodeContains, fieldKey, fieldValue }: UpdateEntityOptions, callback?: () => void) => {
  if (!!nodeContains) {
    cy.contains(nodeId, nodeContains).should('exist').click();
  } else {
    cy.get(nodeId).should('exist').click();
  }

  cy.get(DATA_IDS.DRAWER).should('exist');
  cy.get(DATA_IDS.DRAWER_EDIT).click();

  cy.get(fieldKey).click().focused().clear().type(fieldValue);
  cy.get(fieldKey).should('have.value', fieldValue);

  // The awaits below are an attempt to fix the following flake:
  //
  // CypressError: Timed out retrying after 4050ms: `cy.click()` failed because this element:
  // `<div data-id="Source-2" class="sc-jFQJiD jLKqqL nowheel nodrag">...</div>`
  // is being covered by another element:
  // `<div class="sc-dUwGTt WdrMZ" style="opacity: 0;"></div>`
  //
  // This flake is caused by the fact that the "cancel warning modal" is shown when the user clicks on the "save" or "close" button.
  // This failed to reproduce by user interaction, this could be an issue only for Cypress.

  cy.wait(500).then(() => {
    cy.get(DATA_IDS.DRAWER_SAVE).click();

    cy.wait(500).then(() => {
      cy.get(DATA_IDS.DRAWER_CLOSE).click();

      cy.wait(500).then(() => {
        // press enter to close the warn modal (if any)
        cy.get('body').trigger('keydown', { keyCode: 13 });
        cy.wait(500);
        cy.get('body').trigger('keyup', { keyCode: 13 });

        if (!!callback) callback();
      });
    });
  });
};

interface DeleteEntityOptions {
  nodeId: string;
  nodeContains: string;
  warnModalTitle?: string;
  warnModalNote?: string;
}

export const deleteEntity = ({ nodeId, nodeContains, warnModalTitle, warnModalNote }: DeleteEntityOptions, callback?: () => void) => {
  cy.contains(nodeId, nodeContains).should('exist').click();
  cy.get(DATA_IDS.DRAWER).should('exist');
  cy.get(DATA_IDS.DRAWER_DELETE).click();

  if (!!warnModalTitle) cy.get(DATA_IDS.MODAL).contains(warnModalTitle).should('exist');
  if (!!warnModalNote) cy.get(DATA_IDS.MODAL).contains(warnModalNote).should('exist');

  cy.get(DATA_IDS.APPROVE).click();

  if (!!callback) callback();
};

interface AwaitToastOptions {
  message: string;
}

export const awaitToast = ({ message }: AwaitToastOptions, callback?: () => void) => {
  cy.get(DATA_IDS.TOAST).contains(message).as('toast-msg');
  cy.get('@toast-msg').should('exist');
  cy.get('@toast-msg').parent().parent().find(DATA_IDS.TOAST_CLOSE).click({ force: true });

  if (!!callback) callback();
};

export const handleExceptions = () => {
  return cy.on('uncaught:exception', (err, runnable) => {
    if (err.message.includes('ResizeObserver loop completed with undelivered notifications')) {
      // returning false here prevents Cypress from failing the test
      return false;
    }
    // we still want to ensure there are no other unexpected errors, so we let them fail the test
  });
};
