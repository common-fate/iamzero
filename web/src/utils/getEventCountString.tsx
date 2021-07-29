/** returns a nicely formatted string for the event count */
export const getEventCountString = (eventCount: number) =>
  `${eventCount}  ${eventCount !== 1 ? "events" : "event"}`;
