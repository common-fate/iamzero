import { Flex } from "@chakra-ui/react";
import React from "react";
import { SimpleNavbar } from "./SimpleNavbar";

interface Props {
  withContainer?: boolean;
  children?: React.ReactNode;
}

/**
 * Layout composes SimpleNavbar and justifies children content
 */
const Layout: React.FC<Props> = (props: Props) => {
  // const { logOut } = useAuth();
  return (
    <Flex minH="100vh" w="100vw" overflowX="hidden">
      <SimpleNavbar
        title="iam&#8209;zero"
        navlinks={[
          {
            text: "Alerts",
            path: "/",
          },
        ]}
        // ctaItems={[<Link onClick={logOut}>Log Out</Link>]}
      />
      {/* mt 4.5 is needed to offset the navbar */}
      <Flex
        as="main"
        mt="4.5rem"
        dir="column"
        flexGrow={1}
        maxW="100vw"
        bg="gray.100"
      >
        {props.children}
      </Flex>
    </Flex>
  );
};

export default Layout;
