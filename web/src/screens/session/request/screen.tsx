import type React from 'react'
import { useParams } from 'react-router-dom'

export default function Screen(): React.JSX.Element {
  const { rID } = useParams<Readonly<{ rID: string }>>()

  return (
    <div>
      <h1>Request screen ({rID})</h1>
    </div>
  )
}
