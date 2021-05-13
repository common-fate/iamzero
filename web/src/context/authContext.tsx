import { LockIcon } from "@chakra-ui/icons";
import {
  Button,
  Center,
  FormControl,
  FormErrorMessage,
  Heading,
  HStack,
  Input,
  Stack,
} from "@chakra-ui/react";
import React, { useEffect, useState } from "react";
import { useHistory } from "react-router";
import { checkAuthToken } from "../api";
import { createCtx } from "./createCtx";

export interface AuthContextProps {
  authToken: string;
  logOut: () => void;
}

const [useAuth, AuthContextProvider] = createCtx<AuthContextProps>();

const AuthProvider: React.FC = ({ children }) => {
  const history = useHistory();
  const [authToken, setAuthToken] = useState(
    localStorage.getItem("iamzeroToken")
  );
  const [error, setError] = useState(false);

  // ensure that authToken is saved when it changes
  useEffect(() => {
    if (authToken != null) {
      localStorage.setItem("iamzeroToken", authToken);
    }
  }, [authToken]);

  const [tokenInput, setTokenInput] = useState("");

  // TODO:AUTH
  // will need to be removed as auth is reworked - passing the token as a
  // query param is likely to show up in server logs.
  useEffect(() => {
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get("token");
    if (token) {
      checkAuthToken(token)
        .then((isValid) => {
          if (isValid) {
            setAuthToken(token);
            history.replace("/");
          } else {
            setAuthToken(null);
            setError(true);
          }
        })
        .catch(() => setError(true));
    }
  }, []);

  const onLogin: React.FormEventHandler<HTMLDivElement> = async (e) => {
    e.preventDefault();
    const isValid = await checkAuthToken(tokenInput);
    if (isValid) {
      setAuthToken(tokenInput);
    } else {
      setError(true);
    }
  };

  const logOut = () => setAuthToken(null);

  if (authToken == null) {
    return (
      <Center minH="100vh" w="100vw" bg="gray.100">
        <Stack
          bg="white"
          borderColor="gray.300"
          borderWidth="thin"
          shadow="sm"
          borderRadius={5}
          p={3}
          as="form"
          onSubmit={onLogin}
        >
          <HStack spacing={1}>
            <LockIcon h={3} w={3} />
            <Heading size="md">iam-zero</Heading>
          </HStack>
          <FormControl isInvalid={error === true}>
            <HStack>
              <Input
                w="500px"
                onChange={(e) => setTokenInput(e.target.value)}
                placeholder="Enter authentication token"
              />
              <Button type="submit">Log In</Button>
            </HStack>
            <FormErrorMessage>Invalid token</FormErrorMessage>
          </FormControl>
        </Stack>
      </Center>
    );
  }

  return (
    <AuthContextProvider
      value={{
        authToken,
        logOut,
      }}
    >
      {children}
    </AuthContextProvider>
  );
};

export { useAuth, AuthProvider };
