import { screen } from "@testing-library/react";
import { render } from "../test-utils";
import { createMemoryHistory } from "history";
import { Router } from "react-router";
import Policies from "./Policies";

test("Policies page smoke test", () => {
  const history = createMemoryHistory();
  render(
    <Router history={history}>
      <Policies />
    </Router>
  );
  const linkElement = screen.getByText(/Active/i);
  expect(linkElement).toBeInTheDocument();
});
