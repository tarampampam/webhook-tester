import React, { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useLaseUsedRID } from '~/shared'

export default function Screen(): React.JSX.Element {
  const { rID } = useParams<Readonly<{ rID: string }>>()
  const setLastUsedRID = useLaseUsedRID()[1]

  useEffect((): undefined | (() => void) => {
    if (rID) {
      setLastUsedRID(rID)
    }

    return (): void => {
      setLastUsedRID(undefined)
    }
  }, [rID, setLastUsedRID])

  return (
    <div>
      <h1>Request screen ({rID})</h1>
    </div>
  )
}
