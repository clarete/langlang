/* -*- coding: utf-8; -*-
 *
 * error.h - Macro for error reporting
 *
 * Copyright (C) 2018-2019  Lincoln Clarete
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
#ifndef ERROR_GUARD
#define ERROR_GUARD

#include <stdio.h>              /* for fprintf(), stderr */

/** Report errors that stop the execution of the VM right away */
#define FATAL(f, ...)                                                  \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); exit (EXIT_FAILURE); } \
  while (0)

/** Complain about something and move on */
#define WARN(f, ...)                                                   \
  do { fprintf (stderr, f "\n", ##__VA_ARGS__); }                      \
  while (0)

#endif  /* ERROR_GUARD */
