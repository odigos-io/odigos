import { DATA_IDS } from '../constants';
export * from './cy-alias';

export const visitPage = (path: string, callback?: () => void) => {
  cy.visit(path);

  // Wait for the page to load.
  // On rare occasions, the page might still be blank, and cypress would attempt to interact with it, triggering false errors...
  cy.wait(500).then(() => {
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
  expectedValue: string;
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
  nodeContains: string;
  fieldKey: string;
  fieldValue: string;
}

export const updateEntity = ({ nodeId, nodeContains, fieldKey, fieldValue }: UpdateEntityOptions, callback?: () => void) => {
  cy.contains(nodeId, nodeContains).should('exist').click({ force: true });
  cy.get(DATA_IDS.DRAWER).should('exist');
  cy.get(DATA_IDS.DRAWER_EDIT).click({ force: true });
  cy.get(fieldKey).clear({ force: true }).type(fieldValue);
  cy.get(DATA_IDS.DRAWER_SAVE).click({ force: true });
  cy.get(DATA_IDS.DRAWER_CLOSE).click({ force: true });

  if (!!callback) callback();
};

interface DeleteEntityOptions {
  nodeId: string;
  nodeContains: string;
  warnModalTitle?: string;
  warnModalNote?: string;
}

export const deleteEntity = ({ nodeId, nodeContains, warnModalTitle, warnModalNote }: DeleteEntityOptions, callback?: () => void) => {
  cy.contains(nodeId, nodeContains).should('exist').click({ force: true });
  cy.get(DATA_IDS.DRAWER).should('exist');
  cy.get(DATA_IDS.DRAWER_DELETE).click({ force: true });

  if (!!warnModalTitle) cy.get(DATA_IDS.MODAL).contains(warnModalTitle).should('exist');
  if (!!warnModalNote) cy.get(DATA_IDS.MODAL).contains(warnModalNote).should('exist');

  cy.get(DATA_IDS.APPROVE).click({ force: true });

  if (!!callback) callback();
};

interface AwaitToastOptions {
  withSSE: boolean;
  message: string;
}

export const awaitToast = ({ withSSE, message }: AwaitToastOptions, callback?: () => void) => {
  // In case of SSE, we need around 5 seconds to allow the backend to batch a notification.
  // We will force 2 seconds, and Cypress will add 4 more seconds, giving us 6 seconds total.
  // We don't want to force too much time or we might miss a notification that was sent earlier than expected!

  cy.wait(withSSE ? 2000 : 0).then(() => {
    cy.get(DATA_IDS.TOAST).contains(message).as('toast-msg');
    cy.get('@toast-msg').should('exist');
    cy.get('@toast-msg').parent().parent().find(DATA_IDS.TOAST_CLOSE).click({ force: true });

    if (!!callback) callback();
  });
};
