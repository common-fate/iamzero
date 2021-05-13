/**
 * allows a string or object to be rendered in React without throwing an error.
 * If `value` is an object, it is JSON.stringified.
 */
export const renderStringOrObject = (value: string | Record<string, unknown>) =>
  typeof value === "string" ? value : JSON.stringify(value);
