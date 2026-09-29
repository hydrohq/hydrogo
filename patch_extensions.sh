#!/bin/bash
sed -i '/seen := make(map\[string\]struct{}, len(copyManifests))/a\
	seenPaths := make(map[string]struct{})' internal/adminstudio/extensions.go
sed -i '/seen\[manifest.ID\] = struct{}{}/a\
		for _, surface := range manifest.Surfaces {\n\
			if _, ok := seenPaths[surface.Path]; ok {\n\
				return ExtensionRegistry{}, ErrInvalidAdminExtensionRegistry\n\
			}\n\
			seenPaths[surface.Path] = struct{}{}\n\
		}' internal/adminstudio/extensions.go
