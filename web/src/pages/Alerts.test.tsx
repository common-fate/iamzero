import { screen } from "@testing-library/react";
import { render } from "../test-utils";
import Alerts from "./Alerts";

test("Alerts page smoke test", () => {
  render(<Alerts />);
  const linkElement = screen.getByText(/Active/i);
  expect(linkElement).toBeInTheDocument();
});
