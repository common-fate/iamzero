import {
  Button,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Portal,
  useColorModeValue,
} from "@chakra-ui/react";
import React from "react";
import { Link } from "react-router-dom";

interface Props {
  text: React.ReactNode;
  links: NavLink[];
}

interface NavLink {
  text: React.ReactNode;
  path?: string;
  action?: () => void;
}

const DropdownNavButton = ({ text, links }: Props) => {
  return (
    <Menu>
      <MenuButton
        as={Button}
        variant=""
        fontSize="sm"
        fontWeight="500"
        color={useColorModeValue("gray.700", "whiteAlpha.900")}
        transition="all 0.2s"
        lineHeight="1.5rem"
        height="1.5rem"
      >
        {text}
      </MenuButton>
      <Portal>
        <MenuList zIndex={999} bg="white">
          {links.map((link) => (
            <MenuItem
              as={Link}
              to={link.path || "#"}
              zIndex={999}
              key={link.path}
            >
              {link.text}
            </MenuItem>
          ))}
        </MenuList>
      </Portal>
    </Menu>
  );
};

export default DropdownNavButton;
