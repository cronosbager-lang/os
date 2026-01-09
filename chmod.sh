#!/usr/bin/env bash

set -euo pipefail

echo "Daftar file .sh yang ditemukan:"
echo "--------------------------------"

find . -type f -name "*.sh" -print

echo
echo "Memberikan permission executable (chmod +x)..."
echo "----------------------------------------------"

find . -type f -name "*.sh" -exec chmod +x {} +

echo
echo "Selesai. Semua file .sh sekarang executable."