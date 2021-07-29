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
import AlertRedirectToPolicy from "./pages/AlertRedirectToPolicy";
import Policies from "./pages/Policies";
import PolicyDetails from "./pages/PolicyDetails";
import Tokens from "./pages/Tokens";
import theme from "./theme";

function App() {
  return (
    <AppProviders>
      <Layout>
        <Switch>
          <Route path="/" exact>
            <Redirect to="/policies" />
          </Route>
          <Route path="/policies" exact>
            <Policies />
          </Route>
          <Route path="/policies/:policyId">
            <PolicyDetails />
          </Route>
          <Route path="/tokens" exact>
            <Tokens />
          </Route>
          <Route path="/alerts/:alertId">
            <AlertRedirectToPolicy />
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
