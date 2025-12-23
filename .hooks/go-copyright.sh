#!/bin/bash
set -ex

copyright_header='/*
Copyright © 2025 RedTeamCoin Contributors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/'

echo "Starting copyright check..."

update_copyright() {
	local file="$1"
	local temp_file
	temp_file=$(mktemp)
	echo "${copyright_header}" >"${temp_file}"
	echo "" >>"${temp_file}" # Add an empty line after the header
	sed '/^\/\*/,/^\*\//d' "$file" | sed '/./,$!d' >>"${temp_file}"
	mv "${temp_file}" "${file}"
}

# Get the list of staged .go files
staged_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

# Check if there are any staged .go files
if [[ -z "$staged_files" ]]; then
	echo "No .go files staged for commit. Exiting."
	exit 0
fi

for file in $staged_files; do
	echo "Checking file: $file"
	if grep -qF "Copyright © 2025 RedTeamCoin Contributors" "$file"; then
		echo "Current copyright header is up-to-date in $file"
	else
		echo "Updating copyright header in $file"
		update_copyright "$file"
		echo "Copyright header updated in $file"
	fi
done

echo "Copyright check completed."
