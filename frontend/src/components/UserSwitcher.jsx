export default function UserSwitcher({ crew, currentUser, onChange }) {
  const astronauts = crew.filter((c) => c.persona === 'astronaut')
  const groundControl = crew.filter((c) => c.persona === 'ground-control')

  function handleChange(e) {
    const member = crew.find((c) => c.id === e.target.value)
    if (member) onChange(member)
  }

  return (
    <div className="user-switcher">
      <select value={currentUser?.id ?? ''} onChange={handleChange}>
        {astronauts.length > 0 && (
          <optgroup label="Astronauts">
            {astronauts.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} — {c.role}
              </option>
            ))}
          </optgroup>
        )}
        {groundControl.length > 0 && (
          <optgroup label="Ground Control">
            {groundControl.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} — {c.role}
              </option>
            ))}
          </optgroup>
        )}
      </select>
      {currentUser && (
        <span className="user-mission">
          {currentUser.mission === 'all' ? 'GC' : currentUser.mission}
        </span>
      )}
    </div>
  )
}
