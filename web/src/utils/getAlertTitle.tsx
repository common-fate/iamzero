import { Alert } from "../api-types";

export const getAlertTitle = (alert: Alert) =>
  `${alert.event.data.service}:${alert.event.data.operation}`;
