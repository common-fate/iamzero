// https://chakra-ui.com/docs/theming/customize-theme

import { extendTheme } from "@chakra-ui/react";
import { colors } from "./colors";

const theme = {
  colors: colors,

  config: {
    useSystemColorMode: false,
  },
};

export default extendTheme(theme);
