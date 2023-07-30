export const SETUP = {
  STEPS: {
    CHOOSE_SOURCE: "Choose Source",
    CHOOSE_DESTINATION: "Choose Destination",
    CREATE_CONNECTION: "Create Connection",
    STATUS: {
      ACTIVE: "active",
      DISABLED: "disabled",
      DONE: "done",
    },
    ID: {
      CHOOSE_SOURCE: "choose-source",
      CHOOSE_DESTINATION: "choose-destination",
      CREATE_CONNECTION: "create-connection",
    },
  },
  HEADER: {
    CHOOSE_SOURCE_TITLE: "Select applications to connect",
    CHOOSE_DESTINATION_TITLE: "Add new backend destination from the list",
  },
  MENU: {
    NAMESPACES: "Namespaces",
    SELECT_ALL: "Select All",
    FUTURE_APPLY: "Apply for any future apps",
    TOOLTIP: "Automatically connect any future apps in this namespace",
    SEARCH_PLACEHOLDER: "Search",
    TYPE: "Type",
    MONITORING: "I want to monitor",
  },
  NEXT: "Next",
  BACK: "Back",
  ALL: "All",
  CLEAR_SELECTION: "Clear Selection",
  APPLICATIONS: "Applications",
  RUNNING_INSTANCES: "Running Instances",
  SELECTED: "Selected",
  MANAGED: "Managed",
  CREATE_CONNECTION: "Create Connection",
  UPDATE_CONNECTION: "Update Connection",
  CONNECTION_MONITORS: "This connection will monitor:",
  MONITORS: {
    LOGS: "Logs",
    METRICS: "Metrics",
    TRACES: "Traces",
  },
  DESTINATION_NAME: "Destination Name",
  CREATE_DESTINATION: "Create Destination",
  UPDATE_DESTINATION: "Update Destination",
  QUICK_HELP: "Quick Help",
  ERROR: "Something went wrong",
};

export const INPUT_TYPES = {
  INPUT: "input",
  DROPDOWN: "dropdown",
};

export const OVERVIEW = {
  ODIGOS: "Odigos",
  MENU: {
    OVERVIEW: "Overview",
    SOURCES: "Sources",
    DESTINATIONS: "Destinations",
  },
  ADD_NEW_SOURCE: "Add New Source",
  ADD_NEW_DESTINATION: "Add New Destination",
  DESTINATION_UPDATE_SUCCESS: "Destination updated successfully",
  DESTINATION_CREATED_SUCCESS: "Destination created successfully",
  DESTINATION_DELETED_SUCCESS: "Destination deleted successfully",
  MANAGE: "Manage",
  DELETE: "Delete",
  DELETE_DESTINATION: "Delete Destination",
  DELETE_MODAL_TITLE: "Delete this destination",
  DELETE_MODAL_SUBTITLE:
    "This action cannot be undone. This will permanently delete the destination and all associated data.",
  DELETE_BUTTON: "I want to delete this destination",
};

export const NOTIFICATION = {
  ERROR: "error",
  SUCCESS: "success",
};
