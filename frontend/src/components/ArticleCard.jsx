import { useState } from 'react'
import ReactMarkdown from 'react-markdown'

function scoreClass(score) {
  if (score == null) return 'score-badge'
  if (score >= 0.7) return 'score-badge high'
  if (score < 0.3) return 'score-badge low'
  return 'score-badge'
}

export default function ArticleCard({ article }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="article-card" onClick={() => setExpanded((e) => !e)}>
      <div className="article-card-header">
        <span className="article-id">{article.id}</span>
        <span className="category-badge">{article.category}</span>

        {article.usefulness_score != null && (
          <span className={scoreClass(article.usefulness_score)}>
            {article.usefulness_score.toFixed(2)}
          </span>
        )}

        {article.duplicate_of && (
          <span className="duplicate-badge">dup of {article.duplicate_of}</span>
        )}
      </div>

      <div className="article-title">{article.title}</div>

      {article.tags?.length > 0 && (
        <div className="tags">
          {article.tags.map((t) => (
            <span key={t} className="tag">
              {t}
            </span>
          ))}
        </div>
      )}

      {expanded && (
        <div className="article-body" onClick={(e) => e.stopPropagation()}>
          <ReactMarkdown>{article.body ?? ''}</ReactMarkdown>

          {article.curator_notes && (
            <div className="curator-note">
              <strong>Curator notes:</strong> {article.curator_notes}
            </div>
          )}

          {article.curator_tags?.length > 0 && (
            <div className="tags" style={{ marginTop: 8 }}>
              {article.curator_tags.map((t) => (
                <span key={t} className="curator-tag">
                  {t}
                </span>
              ))}
            </div>
          )}

          {article.last_curated_at && (
            <div className="text-dim" style={{ fontSize: 11, marginTop: 6 }}>
              Last curated: {article.last_curated_at}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
