import { Action } from "../api-types";

export const getAlertTitle = (alert: Action) =>
  `${alert.event.data.service}:${alert.event.data.operation}`;
