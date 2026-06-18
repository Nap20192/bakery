import { createTheme } from '@mui/material/styles';

export const appTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1c1917',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#57534e',
    },
    background: {
      default: '#f7f3ea',
      paper: '#ffffff',
    },
    text: {
      primary: '#1c1917',
      secondary: '#57534e',
    },
  },
  shape: {
    borderRadius: 8,
  },
  typography: {
    fontFamily: '"Roboto", -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif',
    button: {
      textTransform: 'none',
      fontWeight: 600,
    },
  },
  components: {
    MuiButton: {
      defaultProps: {
        disableElevation: true,
      },
      styleOverrides: {
        root: {
          minHeight: 34,
          borderRadius: 6,
        },
        outlined: {
          borderColor: '#a8a29e',
          color: '#292524',
          backgroundColor: '#ffffff',
          '&:hover': {
            borderColor: '#57534e',
            backgroundColor: '#f5f5f4',
          },
        },
        contained: {
          backgroundColor: '#1c1917',
          color: '#ffffff',
          '&:hover': {
            backgroundColor: '#292524',
          },
        },
        text: {
          color: '#57534e',
          '&:hover': {
            backgroundColor: '#f5f5f4',
          },
        },
      },
    },
    MuiInputBase: {
      styleOverrides: {
        root: {
          backgroundColor: '#ffffff',
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          borderRadius: 6,
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
            borderColor: '#1c1917',
            borderWidth: 1,
          },
        },
        notchedOutline: {
          borderColor: '#d6d3d1',
        },
      },
    },
    MuiInputLabel: {
      styleOverrides: {
        root: {
          color: '#57534e',
          '&.Mui-focused': {
            color: '#1c1917',
          },
        },
      },
    },
    MuiMenuItem: {
      styleOverrides: {
        root: {
          minHeight: 36,
          fontSize: 14,
        },
      },
    },
  },
});
