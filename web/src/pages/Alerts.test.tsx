import { screen } from "@testing-library/react";
import { render } from "../test-utils";
import Alerts from "./Alerts";
import { createMemoryHistory } from "history";
import { Router } from "react-router";

test("Alerts page smoke test", () => {
  const history = createMemoryHistory();
  render(
    <Router history={history}>
      <Alerts />
    </Router>
  );
  const linkElement = screen.getByText(/Active/i);
  expect(linkElement).toBeInTheDocument();
});
