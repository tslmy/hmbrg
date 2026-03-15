//go:build mmf

package main

/*
#cgo CFLAGS: -Wno-unused-parameter
#include <string.h>

typedef struct SDL_Window SDL_Window;
typedef struct SDL_Renderer SDL_Renderer;
typedef int SDL_bool;

#define SDL_FALSE 0

typedef struct SDL_GUID {
    unsigned char data[16];
} SDL_GUID;

__attribute__((weak)) SDL_GUID SDL_GUIDFromString(const char *p) {
    SDL_GUID g;
    memset(&g, 0, sizeof(g));
    return g;
}

__attribute__((weak)) void SDL_GUIDToString(SDL_GUID guid, char *dst, int len) {
    (void)guid;
    if (len > 0 && dst != NULL) {
        dst[0] = '\0';
    }
}

__attribute__((weak)) void SDL_ClearComposition(void) {
}

__attribute__((weak)) SDL_bool SDL_IsTextInputShown(void) {
    return SDL_FALSE;
}

__attribute__((weak)) SDL_Window* SDL_RenderGetWindow(SDL_Renderer* renderer) {
    (void)renderer;
    return NULL;
}
*/
import "C"

// This file provides weak SDL symbol stubs for older SDL2 builds used on MMF.
// It is only compiled when the `mmf` build tag is set.
