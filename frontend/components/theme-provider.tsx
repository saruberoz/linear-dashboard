"use client"

import * as React from "react"
import { ThemeProvider as NextThemesProvider } from "next-themes"

// The dashboard uses the light Linear theme only.
function ThemeProvider({
  children,
  ...props
}: React.ComponentProps<typeof NextThemesProvider>) {
  return (
    <NextThemesProvider
      attribute="class"
      forcedTheme="light"
      disableTransitionOnChange
      {...props}
    >
      {children}
    </NextThemesProvider>
  )
}

export { ThemeProvider }
