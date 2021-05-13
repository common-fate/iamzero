import { ChakraProvider, Container } from "@chakra-ui/react";
import React from "react";
import { BrowserRouter as Router, Route, Switch } from "react-router-dom";
import { SWRConfig } from "swr";
import { fetchWithAuth } from "./api";
import { AuthProvider } from "./context/authContext";
import Layout from "./layouts/Layout";
import Alerts from "./pages/Alerts";

function App() {
  return (
    <AppProviders>
      <Layout>
        <Container maxW="1200px" py={5}>
          <Switch>
            <Route path="/" exact>
              <Alerts />
            </Route>
          </Switch>
        </Container>
      </Layout>
    </AppProviders>
  );
}

const AppProviders: React.FC = ({ children }) => {
  return (
    <ChakraProvider>
      <Router>
        <AuthProvider>
          <SWRConfig
            value={{
              fetcher: (resource, init) =>
                fetchWithAuth(resource, init).then((res) => res.json()),
            }}
          >
            {children}
          </SWRConfig>
        </AuthProvider>
      </Router>
    </ChakraProvider>
  );
};

export default App;
