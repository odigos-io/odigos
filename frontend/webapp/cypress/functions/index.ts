export * from './cy-alias';
import { DATA_IDS } from '../constants';

// Apollo may send a single object or a batch array.
const graphqlOperationName = (raw: unknown): string | undefined => {
  if (raw == null) return undefined;
  if (typeof raw === 'string') {
    try {
      return graphqlOperationName(JSON.parse(raw));
    } catch {
      return undefined;
    }
  }
  if (typeof raw !== 'object') return undefined;
  if (Array.isArray(raw)) {
    const first = raw[0] as { operationName?: string } | undefined;
    return first?.operationName;
  }
  return (raw as { operationName?: string }).operationName;
};

// Resolves only when the named operation completes.
// Plain cy.wait('@gql') races with other /graphql traffic (e.g. GetActions after a prior create), which can yield no toast.
export const waitForGraphqlOperation = (operationName: string, maxSkips = 40): Cypress.Chainable => {
  return cy.wait('@gql', { timeout: 120000 }).then((interception) => {
    const op = graphqlOperationName(interception.request.body);
    if (op === operationName) {
      return cy.wrap(interception);
    }
    if (maxSkips <= 0) {
      throw new Error(`waitForGraphqlOperation: expected "${operationName}", got "${op ?? 'unknown'}" (no more skips)`);
    }
    return waitForGraphqlOperation(operationName, maxSkips - 1);
  });
};

export const visitPage = (path: string, callback?: () => void) => {
  cy.visit(path);

  // Wait for the page to load.
  // On rare occasions, the page might still be blank, and cypress would attempt to interact with it, triggering false errors...
  cy.wait(1000).then(() => {
    if (!!callback) callback();
  });
};

interface FindCrdOptions {
  namespace: string;
  crdName: string;
  targetKey: string;
  targetValue: string;
}

// this is not a test, it's a helper function to find a CRD to use in a test
export const findCrdId = ({ namespace, crdName, targetKey, targetValue }: FindCrdOptions, callback: (crdId: string) => void) => {
  const [parentKey, childKey] = targetKey.split('.');

  cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then(({ stdout }) => {
    const crdIds = stdout.split('\n').filter((str) => !!str);

    crdIds.forEach((crdId) => {
      cy.exec(`kubectl get ${crdName} ${crdId} -n ${namespace} -o json`).then(({ stdout }) => {
        const parsed = JSON.parse(stdout);
        const { spec } = parsed?.items?.[0] || parsed || {};
        expect(spec).to.not.be.empty;

        const value = childKey ? spec[parentKey][childKey] : spec[parentKey];
        if (value === targetValue) callback(crdId);
      });
    });
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
  if (!crdId) {
    throw new Error('No CRD ID provided to getCrdById');
  }

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

  cy.get(DATA_IDS.DRAWER_SAVE).click();
  cy.get(DATA_IDS.DRAWER_CLOSE).click();
  if (!!callback) callback();
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
