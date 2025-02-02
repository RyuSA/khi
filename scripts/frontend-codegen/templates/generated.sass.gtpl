// This file is generated by ./scripts/logtype-generator; DO NOT EDIT.
@use "sass:map"
@use "sass:color"

@mixin parent-relationship-label-styles{{range $key,$value := .ParentRelationships}}{{if $value.Visible }}
    &.{{$value.Label}}
        background-color: {{$value.LabelBackgroundColor}}
        color: {{$value.LabelColor}}{{end}}{{end}}

{{range $key,$type := .LogTypes}}
$color-{{$type.Label}}: {{$type.LabelBackgroundColor}}{{end}}

@mixin log-type-shape-colors($log-type-name,$log-type-color,$border-width)
    &.#{$log-type-name}
        background-color: color.adjust($log-type-color, $lightness: 0%)
        &.highlight
            background-color: color.adjust($log-type-color,$lightness: -10%)
        &.selected
            background-color: color.adjust($log-type-color,$lightness: -20%)
        &.dim
            background-color: lightgray

@mixin log-type-shape-colors-for-all
{{range $key,$type := .LogTypes}}   @include log-type-shape-colors("{{$type.Label}}",$color-{{$type.Label}}, 1)
{{end}}

{{range $verkeyb,$value := .Verbs}}
$color-{{$value.CSSSelector | ToLower }}: {{$value.LabelBackgroundColor}}{{end}}

@mixin verb-type-colors($verb-name,$verb-color)
    &.#{$verb-name}
        background-color: $verb-color

@mixin verb-type-colors-for-all
{{range $key,$value := .Verbs}}   @include verb-type-colors("{{$value.CSSSelector | ToLower}}",$color-{{$value.CSSSelector | ToLower}})
{{end}}

{{range $key,$value := .RevisionStates}}
$color-revisionstate-{{$value.CSSSelector}}: {{$value.BackgroundColor}}{{end}}

@mixin revisionstate-type-colors($revision-state-selector,$revision-state-color)
    &.#{$revision-state-selector}
        background-color: $revision-state-color

@mixin revisionstate-type-colors-for-all
{{range $key,$value := .RevisionStates}}   @include revisionstate-type-colors("{{$value.CSSSelector}}",$color-revisionstate-{{$value.CSSSelector | ToLower}})
{{end}}


{{range $key,$value := .Severities}}
$color-severity-{{$value.Label | ToLower}}: {{$value.BackgroundColor}}{{end}}
{{range $key,$value := .Severities}}
$color-severity-{{$value.Label | ToLower}}-label: {{$value.LabelColor}}{{end}}

@mixin log-severity-colors($severity,$color,$label-color)
    &.#{$severity}
        background-color: $color
        color: $label-color

@mixin log-severity-colors-for-all
{{range $key,$value := .Severities}}   @include log-severity-colors("{{$value.Label | ToLower}}",$color-severity-{{$value.Label | ToLower}},$color-severity-{{$value.Label | ToLower}}-label)
{{end}}
