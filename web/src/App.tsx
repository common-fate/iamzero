import { ChakraProvider } from "@chakra-ui/react";
import React from "react";
import {
  BrowserRouter as Router,
  Redirect,
  Route,
  Switch,
} from "react-router-dom";
import { SWRConfig } from "swr";
import { QueryParamProvider } from "use-query-params";
import { fetchWithAuth } from "./api";
import Layout from "./layouts/Layout";
import AlertRedirectToFinding from "./pages/AlertRedirectToPolicy";
import Findings from "./pages/Findings";
import FindingDetails from "./pages/FindingDetails";
import Tokens from "./pages/Tokens";
import theme from "./theme";

function App() {
  return (
    <AppProviders>
      <Layout>
        <Switch>
          <Route path="/" exact>
            <Redirect to="/findings" />
          </Route>
          <Route path="/findings" exact>
            <Findings />
          </Route>
          <Route path="/findings/:findingId">
            <FindingDetails />
          </Route>
          <Route path="/tokens" exact>
            <Tokens />
          </Route>
          <Route path="/alerts/:alertId">
            <AlertRedirectToFinding />
          </Route>
        </Switch>
      </Layout>
    </AppProviders>
  );
}

const AppProviders: React.FC = ({ children }) => {
  return (
    <ChakraProvider theme={theme}>
      <Router>
        <QueryParamProvider ReactRouterRoute={Route}>
          <SWRConfig
            value={{
              fetcher: (resource, init) => fetchWithAuth(resource, init),
            }}
          >
            {children}
          </SWRConfig>
        </QueryParamProvider>
      </Router>
    </ChakraProvider>
  );
};

export default App;
