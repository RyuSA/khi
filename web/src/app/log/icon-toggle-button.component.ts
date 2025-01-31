/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'khi-icon-toggle-button',
  templateUrl: './icon-toggle-button.component.html',
  styleUrls: ['./icon-toggle-button.component.sass'],
  imports: [CommonModule, MatTooltipModule, MatIconModule],
})
export class IconToggleButtonComponent {
  @Input()
  icon = '';

  @Input()
  tooltip = '';

  @Input()
  selected: boolean | null = false;

  @Output()
  selectedChange = new EventEmitter<boolean>();

  @Input()
  disabled: boolean | null = false;

  onClick() {
    this.selectedChange.emit(!this.selected);
  }
}
