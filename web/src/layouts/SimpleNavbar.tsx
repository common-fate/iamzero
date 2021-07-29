import { LockIcon } from "@chakra-ui/icons";
import { chakra, Flex, Heading, Link } from "@chakra-ui/react";
import { NavLink as RouterLink } from "react-router-dom";
import React from "react";
import DropdownNavButton from "./DropdownNavButton";

interface Props {
  //** Optional Title */
  title?: string;
  //** Optional home link, if not, will default to window.location.origin */
  homeLink?: string;
  //** Navlinks */
  navlinks?: NavLink[];
  /**
   * CTA items rendered on the RHS of the nav.
   * Examples could be <Avatar/> or <Button/>
   */
  ctaItems?: React.ReactNode[];
}

interface NavLink {
  text: React.ReactNode;
  path?: string;
  action?: () => void;
  children?: NavLink[];
}

export const SimpleNavbar = (props: Props) => {
  return (
    <chakra.header
      w="full"
      h="4.5rem"
      zIndex={100}
      pos="fixed"
      top="0"
      left="0"
      p={3}
      bg="rgb(27,27,27)"
      color="white"
    >
      <Flex w="100%" h="100%" flexDir="row" alignItems="center">
        {/* Title/logo */}
        <LockIcon mr={1} h={3} w={3} />
        <Link
          href={props.homeLink ? props.homeLink : ""}
          d="flex"
          alignItems="center"
          mr={4}
        >
          {props.title && <Heading size="md">{props.title}</Heading>}
        </Link>

        {/* Nav items */}
        <Flex w="100%" justify="start" align="center">
          {props.navlinks &&
            props.navlinks.length > 0 &&
            props.navlinks.map((navLink, i) => {
              return navLink?.children ? (
                <DropdownNavButton
                  text={navLink.text}
                  links={navLink.children}
                />
              ) : (
                <Link
                  to={navLink.path || "#"}
                  mr={6}
                  key={i}
                  as={RouterLink}
                  activeStyle={{ fontWeight: "bold" }}
                  exact
                >
                  {navLink.text}
                </Link>
              );
            })}
        </Flex>

        {/* CTA Items on RHS */}
        <Flex w="100%" justify="flex-end" alignItems="center">
          {props.ctaItems?.map((ctaItem, i) => (
            <React.Fragment key={i}>{ctaItem}</React.Fragment>
          ))}
        </Flex>
      </Flex>
    </chakra.header>
  );
};
