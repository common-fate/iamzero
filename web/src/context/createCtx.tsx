import React from "react";

/**
 * A helper which creates a React Context with a guard against accessing a
 * Context whose value wasn't provided.
 *
 * [More information](https://github.com/typescript-cheatsheets/react-typescript-cheatsheet#context)
 * */
export function createCtx<A>() {
  const ctx = React.createContext<A | undefined>(undefined);
  function useCtx() {
    const c = React.useContext(ctx);
    if (!c) throw new Error("useCtx must be inside a Provider with a value");
    return c;
  }
  return [useCtx, ctx.Provider] as const; // make TypeScript infer a tuple, not an array of union types
}
