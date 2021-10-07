import { screen } from "@testing-library/react";
import { render } from "../test-utils";
import { createMemoryHistory } from "history";
import { Router } from "react-router";
import Findings from "./Findings";

test("Policies page smoke test", () => {
  const history = createMemoryHistory();
  render(
    <Router history={history}>
      <Findings />
    </Router>
  );
  const linkElement = screen.getByText(/Active/i);
  expect(linkElement).toBeInTheDocument();
});
