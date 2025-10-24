import moment from 'moment'

interface DateFieldProps {
  date: string | null
  format?: string
}

export default function DateField({ date, format = 'MMM DD, YYYY HH:mm' }: DateFieldProps) {
  if (!date) {
    return <span className="text-gray-400">-</span>
  }

  return (
    <span className="text-sm text-gray-700">
      {moment(date).format(format)}
    </span>
  )
}
