import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function combineClasses(...inputs) {
  return twMerge(clsx(inputs))
}
